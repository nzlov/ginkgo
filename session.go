package ginkgo

import (
	"errors"
	"reflect"

	"fmt"

	"sync"

	"strings"

	log "github.com/Sirupsen/logrus"
)

type Session interface {
	method

	ID() string

	Proto() interface{}
	UseProto(remoteService interface{}) error

	Invoke(name string, args []reflect.Value, outTypes []reflect.Type) (results []reflect.Value, err error)
}

type SessionEvent interface {
	OnClientConn(Session, *Conn)
	OnClientClose(Session, *Conn)
	OnClientClear(Session)
}

type session struct {
	methodManager
	sync.Mutex

	proto     interface{}
	protoType reflect.Type

	connWaitGroup sync.WaitGroup
	conns         map[string]*Conn
	id            string
	isRunning     bool
	count         int
	coder         Coder
	sendChan      chan CoderMessage
	reciveMap     map[string]chan CoderMessage
	reciveMutex   sync.Mutex

	event               SessionEvent
	parentMethodManager *methodManager
}

func (s *session) addConn(c *Conn) {
	if s.count > 0 && len(s.conns) >= s.count || !s.isRunning {
		return
	}
	s.Lock()
	if s.event != nil {
		s.event.OnClientConn(s, c)
	}
	c.setSession(s)
	s.conns[c.baseConn.RemoteAddr().String()] = c
	s.connWaitGroup.Add(1)
	go c.start(s.sendChan)
	log.Debugln("Session", "AddConn", c.baseConn.RemoteAddr().String())
	s.Unlock()
}
func (s *session) SetSessionEvent(event SessionEvent) {
	s.event = event
}
func (s *session) ID() string {
	return s.id
}
func (s *session) initSession(id string, n int, coder Coder) *session {
	s.id = id
	s.count = n
	s.isRunning = true
	s.coder = coder
	s.connWaitGroup = sync.WaitGroup{}
	s.conns = make(map[string]*Conn)
	s.sendChan = make(chan CoderMessage, n)
	s.reciveMap = make(map[string]chan CoderMessage)
	s.reciveMutex = sync.Mutex{}
	s.initMethodManager()
	return s
}
func (s *session) setParentMethodManager(manager *methodManager) {
	s.parentMethodManager = manager
}
func (s *session) connclose(c *Conn) {
	s.Lock()
	if s.event != nil {
		s.event.OnClientClose(s, c)
	}
	delete(s.conns, c.baseConn.RemoteAddr().String())
	s.connWaitGroup.Done()
	log.Debugln("Session", "DelConn", c.baseConn.RemoteAddr().String())
	if len(s.conns) == 0 {
		closeMsg := CoderMessage{
			Type: coderMessageType_SessionCloseError,
		}
		for _, r := range s.reciveMap {
			r <- closeMsg
		}
		if s.event != nil {
			s.event.OnClientClear(s)
		}
		s.stop()
	}
	s.Unlock()
}
func (s *session) stop() {
	s.isRunning = false
	for _, v := range s.conns {
		v.stop()
	}
	s.connWaitGroup.Wait()
}

func (s *session) sendmessage(message CoderMessage, n int) {
	if s.isRunning {
		s.sendChan <- message
	}
}

func (s *session) recivemessage(message CoderMessage) {
	switch message.Type {
	case coderMessageType_Invoke:
		message.Type = coderMessageType_InvokeRecive
		name := strings.ToLower(message.Name)
		f, ok := s.RemoteMethods[name]
		if !ok {
			f, ok = s.parentMethodManager.RemoteMethods[name]
		}
		if ok {
			nargs := make([]reflect.Value, len(message.Msg))
			for i, v := range message.Msg {
				nargs[i] = v.Elem()
			}
			ft := f.Function.Type()

			if ft.NumIn() > 1 {
				lat := ft.In(ft.NumIn() - 2)
				switch lat {
				case interfaceType, sessionType:
					nargs = append(nargs, reflect.ValueOf(s))
				case s.protoType:
					nargs = append(nargs, reflect.ValueOf(s.proto))
				case s.protoType.Elem():
					nargs = append(nargs, reflect.ValueOf(s.proto).Elem())
				}
			}

			if ft.NumIn() > 0 {
				lat := ft.In(ft.NumIn() - 1)
				if !ft.IsVariadic() {
					switch lat {
					case interfaceType, sessionType:
						nargs = append(nargs, reflect.ValueOf(s))
					case s.protoType:
						nargs = append(nargs, reflect.ValueOf(s.proto))
					case s.protoType.Elem():
						nargs = append(nargs, reflect.ValueOf(s.proto).Elem())
					}
				}
			}
			r := f.Function.Call(nargs)
			if len(r) > 0 && r[len(r)-1].Type() == errorType {
				rerr := r[len(r)-1]
				if rerr.IsNil() {
					message.Msg = r[:len(r)-1]
				} else {
					message.Type = coderMessageType_InvokeError
					message.Msg = []reflect.Value{reflect.ValueOf(fmt.Sprint(rerr.Elem()))}
				}
			} else {
				message.Msg = r
			}
		} else {
			message.Type = coderMessageType_InvokeNameError
			message.Msg = []reflect.Value{reflect.ValueOf(fmt.Sprintf("Func %s not found", name))}
		}
		s.sendmessage(message, 0)
	default:
		s.reciveMutex.Lock()
		if v, ok := s.reciveMap[message.ID]; ok {
			v <- message
			delete(s.reciveMap, message.ID)
		}
		s.reciveMutex.Unlock()
	}
}

func (s *session) UseProto(remoteService interface{}) error {
	s.proto = remoteService
	s.protoType = reflect.TypeOf(remoteService)

	v := reflect.ValueOf(remoteService)
	if v.Kind() != reflect.Ptr {
		return errors.New("UseService: remoteService argument must be a pointer")
	}
	return buildRemoteService(s, v)
}
func (s *session) Proto() interface{} {
	return s.proto
}

// Invoke the remote method synchronous
func (s *session) Invoke(
	name string,
	args []reflect.Value,
	outTypes []reflect.Type) (results []reflect.Value, err error) {
	//results, err = client.handlerManager.invokeHandler(name, args, context)
	//if results == nil && len(context.ResultTypes) > 0 {
	//	n := len(context.ResultTypes)
	//	results = make([]reflect.Value, n)
	//	for i := 0; i < n; i++ {
	//		results[i] = reflect.New(context.ResultTypes[i]).Elem()
	//	}
	//}
	if !s.isRunning {
		log.Debugln("name", name, "outTypes", len(outTypes), "results", len(results))
		for i := 0; i < len(outTypes); i++ {
			results[i] = reflect.New(outTypes[i]).Elem()
		}
		err = fmt.Errorf("session Close")
		//glog.Debugln("coderMessageType_InvokeNameError", len(results), err)
		return
	}

	id := DefaultUUID.GetID()
	reciveChan := make(chan CoderMessage)
	s.reciveMutex.Lock()
	s.reciveMap[id] = reciveChan
	s.reciveMutex.Unlock()
	s.sendmessage(CoderMessage{
		ID:   id,
		Type: coderMessageType_Invoke,
		Name: name,
		Msg:  args,
	}, 0)
	recives := <-reciveChan
	log.Debugln("recives", recives)
	results = make([]reflect.Value, len(outTypes))
	switch recives.Type {
	case coderMessageType_SessionCloseError:
		for i := 0; i < len(outTypes); i++ {
			results[i] = reflect.New(outTypes[i]).Elem()
		}
		err = fmt.Errorf("%s", "Session Close.")
		return
	case coderMessageType_InvokeNameError:
		for i := 0; i < len(outTypes); i++ {
			results[i] = reflect.New(outTypes[i]).Elem()
		}
		err = fmt.Errorf("%s", recives.Msg[0])
		log.Debugln("coderMessageType_InvokeNameError", len(results), err)
		return
	case coderMessageType_InvokeError:
		for i := 0; i < len(outTypes); i++ {
			results[i] = reflect.New(outTypes[i]).Elem()
		}
		err = fmt.Errorf("%s", recives.Msg[0])
		log.Debugln("coderMessageType_InvokeError", len(results), err)
		return

	}
	args = make([]reflect.Value, 0)
	for i, v := range recives.Msg {
		results[i] = v.Elem()
	}
	//if results == nil && len(outTypes) > 0 {
	//	n := len(recives)
	//	results = make([]reflect.Value, n)
	//	for i := 0; i < n; i++ {
	//		results[i] = reflect.ValueOf(recives[i])
	//	}
	//}
	return
}
