package ginkgo

import "reflect"

var (
	messageHeader = []byte("ginkgo")
	messageInvoke = []byte{0x10}
)

var (
	errorType = reflect.TypeOf((*error)(nil)).Elem()
)
