package main

import (
	"RPC"
	"RPC/registry"
	"RPC/xclient"
	"context"
	"flag"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net"
	"net/http"
	"sync"
	"time"
)

type Foo int

type Args struct{ Num1, Num2 int }

func (f Foo) Sum(args Args, reply *int) error {
	*reply = args.Num1 + args.Num2
	return nil
}

func (f Foo) Sleep(args Args, reply *int) error {
	time.Sleep(time.Second * time.Duration(args.Num1))
	*reply = args.Num1 + args.Num2
	return nil
}

//启动注册中心
//是否传nil 看你自己是否实现了路由注册，如果用路由注册了，那么一定是nil，因为是人家内部的结构再帮你实现
func startRegistry(wg *sync.WaitGroup) {
	//l, _ := net.Listen("tcp", ":9999")
	registry.HandleHTTP() //注册路由
	wg.Done()
	_ = http.ListenAndServe("localhost:9999",nil)
}


//启动服务端，开始监听
func startServer(registryAddr string, wg *sync.WaitGroup) {
	var foo Foo
	l, _ := net.Listen("tcp", "localhost:0")
	server := yyrpc.NewServer()
	_ = server.Register(&foo)
	yyrpc.Heartbeat(registryAddr, "tcp@"+l.Addr().String(), 0) //sendheartbeat没有问题，为什么会这样
	wg.Done()
	server.Accept(l)
}


func foo(xc *xclient.XClient, ctx context.Context, typ, serviceMethod string, args *Args) {
	var reply int
	var err error
	switch typ {
	case "call":
		ctx2,_ := context.WithTimeout(ctx,time.Second*5)
		if err != nil {

		}
		err = xc.Call(ctx2, serviceMethod, args, &reply)
	case "broadcast":
		err = xc.Broadcast(ctx, serviceMethod, args, &reply)
	}
	if err != nil {
		log.Printf("%s %s error: %v", typ, serviceMethod, err)
	} else {
		log.Printf("%s %s success: %d + %d = %d", typ, serviceMethod, args.Num1, args.Num2, reply)
	}
}

func call(registry string) {
	d := xclient.NewGeeRegistryDiscovery(registry, 0)
	xc := xclient.NewXClient(d, xclient.RandomSelect, nil)
	defer func() { _ = xc.Close() }()
	// send request & receive response
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			foo(xc, context.Background(), "call", "Foo.Sum", &Args{Num1: i, Num2: i * i})
		}(i)
	}
	wg.Wait()

}
func broadcast(registry string) {
	d := xclient.NewGeeRegistryDiscovery(registry, 0)
	xc := xclient.NewXClient(d, xclient.RandomSelect, nil)
	defer func() { _ = xc.Close() }()
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			foo(xc, context.Background(), "broadcast", "Foo.Sum", &Args{Num1: i, Num2: i * i})
			// expect 2 - 5 timeout
			// 服务端睡眠了，sleep函数，这是超时控制啊
			ctx, _ := context.WithTimeout(context.Background(), time.Second*2)
			foo(xc, ctx, "broadcast", "Foo.Sleep", &Args{Num1: i, Num2: i * i})
		}(i)
	}
	wg.Wait()
}

func main() {
	//log.SetFlags(0)
	registryAddr := "http://localhost:9999/_geerpc_/registry"
	var wg sync.WaitGroup
	wg.Add(1)
	go startRegistry(&wg)
	wg.Wait()

	time.Sleep(time.Second)
	wg.Add(2)
	go startServer(registryAddr, &wg)
	go startServer(registryAddr, &wg)
	wg.Wait()

	time.Sleep(time.Second)
	call(registryAddr)
	broadcast(registryAddr)
}

func init() {
	lvl := flag.String("level","debug","set the logrus level")
	flag.Parse()
	fmt.Println(*lvl)
	ll,err := log.ParseLevel(*lvl) //处理输出level
	if err != nil {
		ll = log.DebugLevel
	}
	fmt.Println(ll)
	log.SetLevel(ll)
}
