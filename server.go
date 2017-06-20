package ginkgo

import (
	"net"

	"sync"

	log "github.com/Sirupsen/logrus"
)

type ServerEvent interface {
	OnSessionCreate(session Session)
	OnSessionClose(session Session)
}

type ServerOption struct {
}

type TcpServer struct {
	methodManager
	sync.Mutex

	Sessions map[string]*session
	coder    Coder
	event    ServerEvent

	listener *net.TCPListener
}

func NewTcpServer(coder Coder, listener *net.TCPListener) *TcpServer {
	ts := &TcpServer{
		Sessions: make(map[string]*session),
		coder:    coder,
		listener: listener,
	}
	ts.initMethodManager()
	return ts
}
func (ts *TcpServer) SetServerEvent(event ServerEvent) {
	ts.event = event
}

func (ts *TcpServer) Start() {
	var data packet
	for {
		conn, e := ts.listener.Accept()
		if e != nil {
			panic(e)
		}
		go func(c net.Conn) {
			log.Debugln("TcpServer", "NewConn")
			recvData(c, &data)
			log.Debugln("TcpServer", "NewConn", "Data", data)
			message, err := ts.coder.Decoder(data.body)
			if err != nil {
				c.Close()
				return
			}
			if message.Type != coderMessageType_Register {
				c.Close()
				return
			}
			sendData(c, data)
			ts.Lock()
			if _, ok := ts.Sessions[message.ID]; !ok {
				nSession := (&session{}).initSession(message.ID, 0, ts.coder)
				nSession.SetSessionEvent(ts)
				nSession.setParentMethodManager(&ts.methodManager)
				ts.Sessions[message.ID] = nSession
				if ts.event != nil {
					ts.event.OnSessionCreate(nSession)
				}
				log.Infoln("TcpServer", "New Session", message.ID)
			}
			ts.Sessions[message.ID].addConn(NewConn(c))
			ts.Unlock()
		}(conn)
	}
}
func (ts *TcpServer) Stop() {
	ts.listener.Close()
}
func (ts *TcpServer) OnClientConn(s Session, c *Conn)  {}
func (ts *TcpServer) OnClientClose(s Session, c *Conn) {}
func (ts *TcpServer) OnClientClear(s Session) {
	ts.Lock()
	delete(ts.Sessions, s.ID())
	if ts.event != nil {
		ts.event.OnSessionClose(s)
	}
	ts.Unlock()
	log.Infoln("TcpServer", "Del Session", s.ID)
}
