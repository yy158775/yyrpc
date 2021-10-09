package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"time"
	"yyrpc"
	"yyrpc/codec"
)

func startServer(addr chan string) {
	l,err := net.Listen("tcp",":0")
	if err != nil {
		log.Fatal("network error:",err)  //直接终止程序了
	}
	log.Println("start rpc server on",l.Addr())//
	addr <- l.Addr().String() //这个结构
	//server.DefaultServer.Accept(l)
	server.Accept(l) //不用log.Fatal
}

func main() {
	addr := make(chan string)
	go startServer(addr)
	conn,_ := net.Dial("tcp",<-addr) //为什么要写一份重才行
	//defer conn.Close()

	//opt := &server.Option{
	//	MagicNumber: server.MagicNumber,
	//	CodecType:   codec.GobType,
	//}
	//optbyte,_ := json.Marshal(opt)
	//conn.Write(optbyte)
	time.Sleep(time.Second)
	if err := json.NewEncoder(conn).Encode(server.DefaultOption);err != nil {
		fmt.Println("rpc client:encode option error:",err)
	}

	//cc := gob.NewEncoder(conn)
	cc := codec.NewGobCodec(conn)
	for i := 1;i <= 5;i++ {
		h := codec.Header{
			ServiceMethod: "Foo.sum",
			Seq:           int64(i),
			Error:         "",
		}
		if err := cc.Write(&h,fmt.Sprintf("geerpc req %d",h.Seq));err != nil {
			log.Fatal("rpc client:write error:",err)
		}
		if err := cc.ReadHeader(&h);err != nil {
			log.Fatal("rpc client:read header error:",err)
		}
		var reply string
		if err := cc.ReadBody(&reply);err != nil { //必须得用指针的接口才行
			log.Fatal("rpc client:read body error:",err)
		}
		fmt.Println("reply:",reply)
	}
	cc.Close()
}
