package ginkgo

import (
	"net"

	"github.com/Sirupsen/logrus"
)

type ServerOption struct {
}

type TcpServer struct {
	methodManager
	sessions map[string]*session
	coder    Coder

	listener *net.TCPListener
}

func NewTcpServer(coder Coder, listener *net.TCPListener) *TcpServer {
	s := &TcpServer{
		sessions: make(map[string]*session),
		coder:    coder,
		listener: listener,
	}
	s.initMethodManager()
	return s
}

func (s *TcpServer) Start() {
	var data packet
	for {
		c, e := s.listener.Accept()
		if e != nil {
			panic(e)
		}
		logrus.Debugln("TcpServer", "NewConn")
		recvData(c, &data)
		logrus.Debugln("TcpServer", "NewConn", "Data", data)
		message, err := s.coder.Decoder(data.body)
		if err != nil {
			c.Close()
			continue
		}
		if message.Type != coderMessageType_Register {
			c.Close()
			continue
		}
		if _, ok := s.sessions[message.ID]; !ok {
			s.sessions[message.ID] = (&session{}).initSession(0, s.coder)
			s.sessions[message.ID].setParentMethodManager(&s.methodManager)
		}
		s.sessions[message.ID].AddConn(NewConn(c))
		sendData(c, data)
	}
}
func (s *TcpServer) Stop() {
	s.listener.Close()
}
