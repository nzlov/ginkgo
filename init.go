package ginkgo

import (
	"encoding/gob"
	"reflect"
)

func init() {
	gob.Register(&reflect.Value{})
}
