package ginkgo

import (
	"bytes"
	"encoding/binary"
)

func IntToBytes(i int) []byte {
	b_buf := bytes.NewBuffer([]byte{})

	binary.Write(b_buf, binary.BigEndian, i)
	return b_buf.Bytes()
}
func BytesToInt(d []byte) int {
	b_buf := bytes.NewBuffer(d)
	var x int
	binary.Read(b_buf, binary.BigEndian, &x)
	return x
}
