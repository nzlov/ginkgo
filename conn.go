package ginkgo

import (
	"bufio"
	"net"

	log "github.com/Sirupsen/logrus"
)

type conn struct {
	baseConn net.Conn
	session  *session

	isRunning   bool
	messageChan chan CoderMessage
	sendChan    chan CoderMessage
}

func NewConn(c net.Conn) *conn {
	return &conn{
		isRunning:   false,
		baseConn:    c,
		messageChan: make(chan CoderMessage, 10),
		sendChan:    make(chan CoderMessage, 10),
	}
}

func (c *conn) setSession(s *session) {
	c.session = s
}

func (c *conn) start() {
	c.isRunning = true
	go func() {
		defer func() {
			c.stop()
		}()
		reader := bufio.NewReader(c.baseConn)
		var data packet
		var message CoderMessage
		var err error
		for c.isRunning {
			log.Debugln("Conn", "Recive", "Wait")
			if err := recvData(reader, &data); err != nil {
				break
			}
			log.Debugln("Conn", "Recive", data)
			message, err = c.session.coder.Decoder(data.body)
			if err != nil {
				continue
			}
			c.session.recivemessage(message)
		}
	}()
	log.Debugln("Conn", "Start")
	for m := range c.sendChan {
		if c.isRunning {
			log.Debugln("Conn", "SendChan", m)
			data := c.session.coder.Encoder(m)
			err := sendData(c.baseConn, packet{
				body:       data,
				fullDuplex: true,
			})

			if err != nil {
				//glog.NewTagField("conn").Errorln("send", err)
				c.stop()
				return
			}
		} else {
			c.session.sendmessage(m, 0)
		}
	}
	c.baseConn.Close()
	c.session.connclose(c)
	log.Debugln("Conn", "Stop")
}
func (c *conn) stop() {
	if !c.isRunning {
		return
	}
	c.isRunning = false
	close(c.sendChan)
}

func (c *conn) send(message CoderMessage) bool {
	if c.isRunning {
		c.sendChan <- message
	}
	return c.isRunning
}

//
//func (c *conn) invoke() ([]byte, bool) {
//	if !c.isRunning {
//		return nil, false
//	}
//	data := make([]byte, 0)
//	buf := bytes.NewBuffer(data)
//	buf.Write(messageHeader)
//	buf.Write(messageInvoke)
//	id := DefaultUUID.GetID()
//	buf.Write([]byte(id))
//	buf.Write(IntToBytes(len(d)))
//	buf.Write(d)
//	c.send <- buf.Bytes()
//	rd, ok := <-c.message
//	return rd, ok
//}
