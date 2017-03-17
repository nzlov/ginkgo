package ginkgo

import (
	"encoding/gob"
	"net"

	"errors"

	"github.com/nzlov/glog"
)

type coderMessageType int

const (
	coderMessageType_Unknown = iota
	coderMessageType_Binary
	coderMessageType_Invoke
	coderMessageType_InvokeRecive
)

type CoderMessage struct {
	ID   string
	Type coderMessageType
	Msg  []interface{}
}

type Coder interface {
	GetCoder(conn net.Conn) Coder
	Encoder(message CoderMessage) error
	Decoder() (CoderMessage, error)
}

type GobCoder struct {
	e *gob.Encoder
	d *gob.Decoder
}

func NewGobCoder() *GobCoder {
	return &GobCoder{}
}

func (g *GobCoder) GetCoder(conn net.Conn) Coder {
	return &GobCoder{
		e: gob.NewEncoder(conn),
		d: gob.NewDecoder(conn),
	}
}
func (g *GobCoder) Encoder(message CoderMessage) error {
	glog.Debugln("GobCoder", "Encoder", message)
	err := g.e.Encode(message)
	return err
}

func (g *GobCoder) Decoder() (CoderMessage, error) {
	c := CoderMessage{}
	err := g.d.Decode(&c)
	if c.Type == coderMessageType_Unknown {
		return CoderMessage{}, errors.New("client exit")
	}
	return c, err
}
