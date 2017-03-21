package ginkgo

import (
	"errors"
	"reflect"
)

func buildRemoteService(s *Session, v reflect.Value) error {
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
				buildRemoteMethod(s, f, ft, sf)
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

func buildRemoteMethod(s *Session,
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
		fn = getSyncRemoteMethod(s, name, outTypes, ft.IsVariadic(), hasError)
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

func getSyncRemoteMethod(s *Session,
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

func callService(function reflect.Value,
	name string, args []reflect.Value) (results []reflect.Value, err error) {
	//if context.IsMissingMethod() {
	//	missingMethod := function.Interface().(MissingMethod)
	//	return missingMethod(name, args, context)
	//}
	ft := function.Type()
	if !ft.IsVariadic() {
		count := len(args)
		n := ft.NumIn()
		if n < count {
			args = args[:n]
		}
	}
	results = function.Call(args)
	n := ft.NumOut()
	if n == 0 {
		return results, nil
	}
	if ft.Out(n-1) == errorType {
		err, _ = results[n-1].Interface().(error)
		results = results[:n-1]
	}
	return results, err
}
