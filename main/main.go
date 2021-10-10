package main

import (
	"fmt"
	"log"
	"net"
	"sync"
	"yyrpc/client"

	//"time"
	"yyrpc"
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
	client, err := client.Dial("tcp", <-addr)
	if err != nil {
		log.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 1; i <= 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			args := fmt.Sprintf("yyrpc seq:%d", i)
			var reply string
			if err := client.Call("foo.sum", args, &reply); err != nil {
				log.Fatal("call foo.sum error:", err)
			}
			log.Println("reply:", reply)
		}(i)
	}
	wg.Wait()
}