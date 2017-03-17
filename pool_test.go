package ginkgo

import (
	"fmt"
	"net"
	"testing"
)

func TestPool(t *testing.T) {
	p := NewPool(10)
	pl := len(p.pool)
	pc := cap(p.pool)

	if pl != 10 {
		t.Error("NewPool", 10, "len(p.pool)", pl, "!= 10")
	}
	if pc != 10 {
		t.Error("NewPool", 10, "cap(p.pool)", pc, "!= 10")
	}

	c, err := net.Dial("tcp", "127.0.0.1:6020")
	if err != nil {
		fmt.Println("Error", err)
		return
	}
	p.Put(c)
	p.Put(c)
	p.Put(c)
	p.Put(c)
	p.Put(c)
	fmt.Println(p.pool)
	fmt.Println(p.Get(), p.pool)
	fmt.Println(p.Get(), p.pool)
	fmt.Println(p.Get(), p.pool)
	fmt.Println(p.Get(), p.pool)
	fmt.Println(p.Get(), p.pool)
	fmt.Println(p.Get(), p.pool)
	fmt.Println(p.Get(), p.pool)
	fmt.Println(p.Get(), p.pool)
	fmt.Println(p.Get(), p.pool)

}
