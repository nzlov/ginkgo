package ginkgo

import (
	"fmt"
	"strings"
	"time"
)

var DefaultUUID = newUUID()

type uuid struct {
	idindex int
	otime   string
}

func newUUID() *uuid {
	return &uuid{
		idindex: 0,
	}
}
func (u *uuid) start() *uuid {

	return u
}

func (u *uuid) GetID() string {
	ntime := fmt.Sprint(time.Now().Unix())
	if u.otime == ntime {
		u.idindex++
	} else {
		u.otime = ntime
		u.idindex = 0
	}
	i := fmt.Sprint(u.idindex)
	ntime = ntime + strings.Repeat("0", 5-len(i)) + i
	return ntime
}
