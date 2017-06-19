package main

import (
	"net"
	"os"

	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/nzlov/ginkgo"
)

func add(a, b int) int {
	return a + b
}
func chu(a, b int) (int, error) {
	if b == 0 {
		return 0, fmt.Errorf("b is not zero!")
	}
	return a / b, nil
}
func swap(a, b int, s ginkgo.Session) (int, int) {
	log.Infoln("Session", s.ID(), "swap", a, b)
	return b, a
}
func panitln(a string) {

	// glog.Infoln(a)
}
func init() {
	// Log as JSON instead of the default ASCII formatter.
	log.SetFormatter(&log.TextFormatter{})

	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	log.SetOutput(os.Stdout)

	// Only log the warning severity or above.
	log.SetLevel(log.DebugLevel)
}

type Proto struct {
	Test func(int, int) int
}

var protos map[string]Proto

type Event struct {
}

func (e *Event) OnSessionCreate(session ginkgo.Session) {
	proto := Proto{}
	session.UseProto(&proto)
	protos[session.ID()] = proto
	session.AddFunction("add", add)
}

func (e *Event) OnSessionClose(session ginkgo.Session) {

}

func main() {
	// glog.Start()
	// defer glog.Close()
	// glog.Register(colorconsole.New(colorconsole.ShowTypeDefault))
	// glog.SetLevel(glog.DebugLevel)
	protos = make(map[string]Proto)

	listener, err := net.Listen("tcp", "0.0.0.0:4444")
	if err != nil {
		panic(err)
	}
	server := ginkgo.NewTcpServer(ginkgo.NewHproseCoder(), listener.(*net.TCPListener))
	server.AddFunction("swap", swap)
	server.AddFunction("panitln", panitln)
	server.AddFunction("chu", chu)

	server.SetServerEvent(&Event{})

	server.Start()

}
