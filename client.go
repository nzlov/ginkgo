package ginkgo

import (
	"fmt"
	"net"
	"reflect"
)

type ClientOption struct {
}

type Client struct {
	Session
	host  string
	coder Coder
	n     int
	id    string
}

func NewClient(host string, n int, coder Coder) *Client {
	return &Client{
		host:  host,
		n:     n,
		id:    DefaultUUID.GetID(),
		coder: coder,
	}
}

func (c *Client) Start() error {
	c.initSession(c.id, c.n, c.coder)
	var data packet
	for i := 0; i < c.n; i++ {
		conn, err := net.Dial("tcp", c.host)
		if err != nil {
			return err
		}
		sendData(conn, packet{
			body: c.coder.Encoder(CoderMessage{
				ID:   c.id,
				Type: coderMessageType_Register,
				Name: c.id,
				Msg:  []reflect.Value{},
			}),
			fullDuplex: true,
		})
		recvData(conn, &data)
		message, err := c.coder.Decoder(data.body)
		if err != nil {
			return err
		}
		if message.Type != coderMessageType_Register {
			return fmt.Errorf("Conn register lose!")
		}
		conns := NewConn(conn)
		c.AddConn(conns)
	}
	return nil
}
