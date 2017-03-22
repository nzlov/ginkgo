package ginkgo

import (
	"bufio"
	"net"

	log "github.com/Sirupsen/logrus"
)

type Conn struct {
	baseConn net.Conn
	Session  *Session

	isRunning bool

	exit chan bool
}

func NewConn(c net.Conn) *Conn {
	return &Conn{
		isRunning: false,
		baseConn:  c,
		exit:      make(chan bool),
	}
}

func (c *Conn) setSession(s *Session) {
	c.Session = s
}

func (c *Conn) start(sendChan chan CoderMessage) {
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
			message, err = c.Session.coder.Decoder(data.body)
			if err != nil {
				continue
			}
			go c.Session.recivemessage(message)
		}
	}()
	log.Debugln("Conn", "Start")
	for c.isRunning {
		select {
		case <-c.exit:
			break
		case m := <-sendChan:
			if c.isRunning {
				log.Debugln("Conn", "SendChan", m)
				data := c.Session.coder.Encoder(m)
				err := sendData(c.baseConn, packet{
					body:       data,
					fullDuplex: true,
				})
				if err != nil {
					//glog.NewTagField("Conn").Errorln("send", err)
					sendChan <- m
					c.stop()
					break
				}
			} else {
				sendChan <- m
				break
			}
		}
	}
	c.baseConn.Close()
	c.Session.connclose(c)
	log.Debugln("Conn", "Stop")
}
func (c *Conn) stop() {
	if !c.isRunning {
		return
	}
	c.isRunning = false
	c.exit <- true
}

//
//func (c *Conn) invoke() ([]byte, bool) {
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
