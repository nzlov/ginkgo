package ginkgo

import (
	"errors"
	"reflect"

	"fmt"

	"sync"

	"time"

	"strings"

	log "github.com/Sirupsen/logrus"
)

type session struct {
	methodManager
	sync.Mutex
	num int

	isRunning bool
	count     int
	pool      *pool
	coder     Coder
	sendChan  chan CoderMessage
	reciveMap map[string]chan CoderMessage

	parentMethodManager *methodManager
}

func NewSession() session {
	return session{}
}

func (s *session) AddConn(c *conn) {
	c.setSession(s)
	b := s.pool.Add(c)
	if b {
		go c.start()
		s.num++
	} else {
		c.setSession(nil)
		c.stop()
	}
}
func (s *session) initSession(n int, coder Coder) *session {
	s.num = 0
	s.isRunning = true
	s.coder = coder
	s.pool = newPool(n)
	s.count = n
	s.sendChan = make(chan CoderMessage, n)
	s.reciveMap = make(map[string]chan CoderMessage)
	s.initMethodManager()
	return s
}
func (s *session) setParentMethodManager(manager *methodManager) {
	s.parentMethodManager = manager
}
func (s *session) connclose(c *conn) {
	s.pool.Remove(c)
}
func (s *session) Stop() {
	s.isRunning = false
	for s.pool.num > 0 {
		p := s.pool.Get()
		if p == nil {
			return
		}
		p.stop()
		s.pool.Put(p)
	}
	log.Debugln("session", "pool", "num", s.pool.num)
}

func (s *session) sendmessage(message CoderMessage, n int) {
	if !s.isRunning || n > 10 {
		return
	}
	c := s.pool.Get()
	if c == nil || !c.isRunning {
		time.Sleep(time.Microsecond)
		s.sendmessage(message, n+1)
		return
	}
	c.send(message)
	s.pool.Put(c)

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
			r := f.Function.Call(nargs)
			message.Msg = r
		} else {

			message.Type = coderMessageType_InvokeNameError
			message.Msg = []reflect.Value{reflect.ValueOf(fmt.Sprintf("Func %s not found", name))}
		}
		s.sendmessage(message, 0)
	case coderMessageType_InvokeRecive:
		fallthrough
	case coderMessageType_InvokeNameError:
		s.Lock()
		if v, ok := s.reciveMap[message.ID]; ok {
			v <- message
			delete(s.reciveMap, message.ID)
		}
		s.Unlock()
	}
}

func (s *session) UseProto(remoteService interface{}) error {
	v := reflect.ValueOf(remoteService)
	if v.Kind() != reflect.Ptr {
		return errors.New("UseService: remoteService argument must be a pointer")
	}
	return buildRemoteService(s, v)
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

	id := DefaultUUID.GetID()
	reciveChan := make(chan CoderMessage)
	s.Lock()
	s.reciveMap[id] = reciveChan
	s.Unlock()
	s.sendmessage(CoderMessage{
		ID:   id,
		Type: coderMessageType_Invoke,
		Name: name,
		Msg:  args,
	}, 0)
	recives := <-reciveChan
	//glog.Debugln("recives", recives)
	results = make([]reflect.Value, len(outTypes))
	switch recives.Type {
	case coderMessageType_InvokeNameError:
		for i := 0; i < len(outTypes); i++ {
			results[i] = reflect.New(outTypes[i]).Elem()
		}
		err = fmt.Errorf("%s", recives.Msg[0])
		//glog.Debugln("coderMessageType_InvokeNameError", len(results), err)
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
