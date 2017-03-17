package ginkgo

import (
	"errors"
	"reflect"

	hio "github.com/hprose/hprose-golang/io"
)

type session struct {
	pool      chan *conn
	coder     Coder
	sendChan  chan CoderMessage
	reciveMap map[string]chan []interface{}
}

func NewSession(n int, coder Coder) *session {
	return &session{
		pool:      make(chan *conn, n),
		coder:     coder,
		sendChan:  make(chan CoderMessage, n),
		reciveMap: make(map[string]chan []interface{}),
	}
}

func (s *session) AddConn(c *conn) {
	c.setSession(s)
	go c.start()
	s.pool <- c
}

func (s *session) sendmessage(message CoderMessage) {
	c := <-s.pool
	if ok := c.send(message); !ok {
		s.sendmessage(message)
	}
	s.pool <- c
}

func (s *session) recivemessage(message CoderMessage) {
	switch message.Type {
	case coderMessageType_Invoke:
		message.Type = coderMessageType_InvokeRecive
		message.Msg = []interface{}{3}
		s.sendmessage(message)
	case coderMessageType_InvokeRecive:
		if v, ok := s.reciveMap[message.ID]; ok {
			v <- message.Msg
			delete(s.reciveMap, message.ID)
		}
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
	reciveChan := make(chan []interface{})
	s.reciveMap[id] = reciveChan
	s.sendmessage(CoderMessage{
		ID:   id,
		Type: coderMessageType_Invoke,
		Msg:  body,
	})
	recives := <-reciveChan
	if results == nil && len(outTypes) > 0 {
		n := len(recives)
		results = make([]reflect.Value, n)
		for i := 0; i < n; i++ {
			results[i] = reflect.ValueOf(recives[i])
		}
	}
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
