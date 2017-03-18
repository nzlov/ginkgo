package ginkgo

import "net"

type conn struct {
	baseConn net.Conn
	session  *session
	coder    Coder

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
	c.coder = s.coder.GetCoder(c.baseConn)
}

func (c *conn) start() {
	c.isRunning = true
	go func() {
		defer func() {
			c.stop()
		}()
		for c.isRunning {
			//glog.Debugln("Conn", "Recive", "Wait")
			n, err := c.coder.Decoder()
			if err != nil {
				//glog.NewTagField("conn").Errorln("reicve", err)
				return
			}
			//glog.Debugln("Conn", "Recive", n)
			c.session.recivemessage(n)
		}
	}()
	//glog.Debugln("Conn", "Start")
	for m := range c.sendChan {
		if c.isRunning {
			err := c.coder.Encoder(m)
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
