package main

import (
	yyrpc "RPC"
	log "github.com/sirupsen/logrus"
	"net"
	"time"
)

type Foo int

type SumArgs struct {
	A int
	B int
}

func (f *Foo) Sum(arg SumArgs,reply *int) error {
	*reply = arg.A + arg.B
	return nil
}


func (f *Foo) HelloWorld(args string,reply *string) error {
	*reply = "Hello" + args
	return nil
}

var registry string = "http://localhost:9091/registry"

func main() {
	server := yyrpc.NewServer()
	_ = server.Register(new(Foo))

	listener,err := net.Listen("tcp","localhost:0")
	if err != nil {
		log.Fatal("rpc serevr:listen error:",err)
	}
	log.Info("Listen addr:",listener.Addr().String())

	err = yyrpc.Heartbeat(registry,"tcp@" + listener.Addr().String(),time.Minute*5-1)

	if err != nil {
		log.Fatal("rpc serevr:heartbeat error:",err)
	}
	server.Accept(listener)
}
