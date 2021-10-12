package main

import (
	"log"
	"net"
	"sync"
	"time"
	//"time"
	"yyrpc"
)

type Foo int

type Args struct{ Num1, Num2 int }

func (f Foo) Sum(args Args, reply *int) error {
	*reply = args.Num1 + args.Num2
	return nil
}

//主要是服务端调用时候，肯定知道是哪个函数，凡是服务端进行调用的时候，就需要在注册时，讲方法全部处理好了
//

func startServer(addr chan string) {
	var foo Foo
	if err := yyrpc.Register(&foo); err != nil {
		log.Fatal("register error:", err)
	}
	// pick a free port
	l, err := net.Listen("tcp", ":0")

	if err != nil {
		log.Fatal("network error:", err)
	}

	log.Println("start rpc server on", l.Addr())

	addr <- l.Addr().String()

	yyrpc.Accept(l)
}

func main() {
	log.SetFlags(0)
	addr := make(chan string)
	go startServer(addr)

	client, _ := yyrpc.Dial("tcp", <-addr)

	defer func() { _ = client.Close() }()

	time.Sleep(time.Second)
	// send request & receive response
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			args := &Args{Num1: i, Num2: i * i}
			var reply int

			if err := client.Call("Foo.Sum", args, &reply); err != nil {
				log.Fatal("call Foo.Sum error:", err)
			}

			log.Printf("%d + %d = %d", args.Num1, args.Num2, reply)
		}(i)
	}
	wg.Wait()
}