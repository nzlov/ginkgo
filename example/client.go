package main

import (
	"os"

	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/nzlov/ginkgo"
)

type Proto struct {
	Add     func(int, int) int
	Swap    func(int, int) (int, int, error)
	Panitln func(string) error
	Chu     func(int, int) (int, error)
}

func test(a, b int) int {
	return a * b
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

func main() {
	client := ginkgo.NewClient("0.0.0.0:4444", 10, ginkgo.NewHproseCoder())

	var p Proto
	client.AddFunction("test", test)
	client.UseProto(&p)
	// glog.Infoln("1+2=", p.Add(&T{1, 2}), time.Now().Sub(t))

	err := client.Start()
	if err != nil {
		log.Fatalln(err)
	}

	log.Println(p.Add(2, 2))
	a, b, e := p.Swap(1, 2)
	if e != nil {
		log.Errorln("Panitln", a, b, e)
	}
	log.Infoln("1 2 swap", a, b)
	e = p.Panitln("test")
	if e != nil {
		panic(e)
	}

	log.Println(p.Chu(2, 2))
	log.Println(p.Chu(2, 0))
	log.Println(p.Chu(2, 0))

	//log.Println("Exit")

	count := 0
	schan := make(chan bool)
	exit := make(chan bool)
	w := sync.WaitGroup{}
	isrunning := true
	go func() {
		for _ = range schan {
			p.Add(1, 2)
			count++
		}
	}()
	go func() {
		for isrunning {
			select {
			case <-exit:
				close(schan)
				log.Infoln("Count:", count, time.Second/time.Duration(count))
				isrunning = false
				w.Done()
			case schan <- true:
			}
		}
	}()
	w.Add(1)
	<-time.After(time.Second)
	exit <- true
	w.Wait()
}
