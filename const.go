package ginkgo

import "reflect"

var (
	messageHeader = []byte("ginkgo")
	messageInvoke = []byte{0x10}
)

var (
	errorType     = reflect.TypeOf((*error)(nil)).Elem()
	interfaceType = reflect.TypeOf((*interface{})(nil)).Elem()
	sessionType   = reflect.TypeOf((*Session)(nil))
)
