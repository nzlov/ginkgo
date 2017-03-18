package ginkgo

import (
	"encoding/gob"
	"errors"
	"reflect"
)

func init() {
	gob.Register(&reflect.Value{})
	gob.Register(errors.New("").Error)
}
