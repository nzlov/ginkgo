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

	//exit chan bool
}

func NewClientWithID(host string, id string, n int, coder Coder) *Client {
	c := &Client{
		host:  host,
		n:     n,
		id:    id,
		coder: coder,
		//exit:  make(chan bool),
	}
	c.initSession(c.id, c.n, c.coder)
	//c.SetSessionEvent(c)
	return c
}
func NewClient(host string, n int, coder Coder) *Client {
	c := &Client{
		host:  host,
		n:     n,
		id:    DefaultUUID.GetID(),
		coder: coder,
		//exit:  make(chan bool),
	}
	c.initSession(c.id, c.n, c.coder)
	//c.SetSessionEvent(c)
	return c
}

func (c *Client) Start() error {
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

func (c *Client) ID() string {
	return c.id
}
