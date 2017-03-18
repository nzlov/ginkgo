package ginkgo

import (
	"errors"
	"reflect"

	"fmt"

	"sync"

	"time"

	hio "github.com/hprose/hprose-golang/io"
)

type session struct {
	isRunning bool
	count     int
	pool      *pool
	coder     Coder
	sendChan  chan CoderMessage
	reciveMap map[string]chan CoderMessage
	function  map[string]interface{}
	mutex     sync.Mutex
}

func NewSession(n int, coder Coder) *session {
	return &session{
		isRunning: true,
		count:     n,
		pool:      newPool(n),
		coder:     coder,
		sendChan:  make(chan CoderMessage, n),
		reciveMap: make(map[string]chan CoderMessage),
		function:  make(map[string]interface{}),
		mutex:     sync.Mutex{},
	}
}

func (s *session) AddConn(c *conn) {
	c.setSession(s)
	b := s.pool.Add(c)
	if b {
		go c.start()
	}
}
func (s *session) Start() {
	s.isRunning = true
}
func (s *session) connclose(c *conn) {
	s.pool.Remove(c)
}
func (s *session) AddFunction(name string, f interface{}) {
	s.function[name] = f
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
	//glog.Debugln("session", "pool", "num", s.pool.num)
}

func (s *session) sendmessage(message CoderMessage, n int) {
	if !s.isRunning || n > 10 {
		return
	}
	c := s.pool.Get()
	//glog.Debugln("Session", "sendmessage", n, c)
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
		name := message.Msg[0].(string)
		args := make([]reflect.Value, 0)
		hio.Unmarshal(message.Msg[1].([]byte), &args)
		if f, ok := s.function[name]; ok {
			fv := reflect.ValueOf(f)
			nargs := make([]reflect.Value, len(args))
			for i, v := range args {
				nargs[i] = v.Elem()
			}
			r := fv.Call(nargs)
			message.Msg = []interface{}{hio.Marshal(r)}
		} else {
			message.Type = coderMessageType_InvokeNameError
			message.Msg = []interface{}{fmt.Sprintf("Func %s not found", name)}
		}
		s.sendmessage(message, 0)
	case coderMessageType_InvokeRecive:
		fallthrough
	case coderMessageType_InvokeNameError:
		s.mutex.Lock()
		if v, ok := s.reciveMap[message.ID]; ok {
			v <- message
			delete(s.reciveMap, message.ID)
		}
		s.mutex.Unlock()
	}
}

func (s *session) UseProto(remoteService interface{}) error {
	v := reflect.ValueOf(remoteService)
	if v.Kind() != reflect.Ptr {
		return errors.New("UseService: remoteService argument must be a pointer")
	}
	return s.buildRemoteService(v)
}

func (s *session) buildRemoteService(v reflect.Value) error {
	v = v.Elem()
	t := v.Type()
	et := t
	if et.Kind() == reflect.Ptr {
		et = et.Elem()
	}
	ptr := reflect.New(et)
	obj := ptr.Elem()
	count := obj.NumField()
	for i := 0; i < count; i++ {
		f := obj.Field(i)
		ft := f.Type()
		sf := et.Field(i)
		if ft.Kind() == reflect.Ptr {
			ft = ft.Elem()
		}
		if f.CanSet() {
			switch ft.Kind() {
			case reflect.Struct:
				//s.buildRemoteSubService(f, ft, sf)
			case reflect.Func:
				s.buildRemoteMethod(f, ft, sf)
			}
		}
	}
	if t.Kind() == reflect.Ptr {
		v.Set(ptr)
	} else {
		v.Set(obj)
	}
	return nil
}

func (s *session) buildRemoteMethod(
	f reflect.Value, ft reflect.Type, sf reflect.StructField) error {
	name := getRemoteMethodName(sf)
	outTypes, hasError := getResultTypes(ft)
	async := false
	if outTypes == nil && hasError == false {
		if ft.NumIn() > 0 && ft.In(0).Kind() == reflect.Func {
			cbft := ft.In(0)
			if cbft.IsVariadic() {
				return errors.New("callback can't be variadic function")
			}
			outTypes, hasError = getCallbackResultTypes(cbft)
			async = true
		}
	}

	var fn func(in []reflect.Value) (out []reflect.Value)
	if async {
		//fn = s.getAsyncRemoteMethod(name, outTypes, ft.IsVariadic(), hasError)
	} else {
		fn = s.getSyncRemoteMethod(name, outTypes, ft.IsVariadic(), hasError)
	}
	if f.Kind() == reflect.Ptr {
		fp := reflect.New(ft)
		fp.Elem().Set(reflect.MakeFunc(ft, fn))
		f.Set(fp)
	} else {
		f.Set(reflect.MakeFunc(ft, fn))
	}
	return nil
}

func (s *session) getSyncRemoteMethod(
	name string,
	outTypes []reflect.Type,
	isVariadic, hasError bool) func(in []reflect.Value) (out []reflect.Value) {
	return func(in []reflect.Value) (out []reflect.Value) {
		if isVariadic {
			in = getIn(in)
		}
		var err error
		out, err = s.Invoke(name, in, outTypes)
		if hasError {
			out = append(out, reflect.ValueOf(&err).Elem())
		} else if err != nil {
			panic(err)
		}
		return
	}
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

	body := make([]interface{}, 2)
	body[0] = name
	body[1] = hio.Marshal(args)

	id := DefaultUUID.GetID()
	reciveChan := make(chan CoderMessage)
	s.mutex.Lock()
	s.reciveMap[id] = reciveChan
	s.mutex.Unlock()
	s.sendmessage(CoderMessage{
		ID:   id,
		Type: coderMessageType_Invoke,
		Msg:  body,
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
	hio.Unmarshal(recives.Msg[0].([]byte), &args)
	for i, v := range args {
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

func getRemoteMethodName(sf reflect.StructField) (name string) {
	name = sf.Tag.Get("name")
	if name == "" {
		name = sf.Name
	}
	return
}

func getCallbackResultTypes(ft reflect.Type) ([]reflect.Type, bool) {
	n := ft.NumIn()
	if n == 0 {
		return nil, false
	}
	hasError := (ft.In(n-1) == errorType)
	if hasError {
		n--
	}
	results := make([]reflect.Type, n)
	for i := 0; i < n; i++ {
		results[i] = ft.In(i)
	}
	return results, hasError
}

func getResultTypes(ft reflect.Type) ([]reflect.Type, bool) {
	n := ft.NumOut()
	if n == 0 {
		return nil, false
	}
	hasError := (ft.Out(n-1) == errorType)
	if hasError {
		n--
	}
	results := make([]reflect.Type, n)
	for i := 0; i < n; i++ {
		results[i] = ft.Out(i)
	}
	return results, hasError
}

func getIn(in []reflect.Value) []reflect.Value {
	inlen := len(in)
	varlen := 0
	argc := inlen - 1
	varlen = in[argc].Len()
	argc += varlen
	args := make([]reflect.Value, argc)
	if argc > 0 {
		for i := 0; i < inlen-1; i++ {
			args[i] = in[i]
		}
		v := in[inlen-1]
		for i := 0; i < varlen; i++ {
			args[inlen-1+i] = v.Index(i)
		}
	}
	return args
}
