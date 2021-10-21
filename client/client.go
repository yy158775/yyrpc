package main

import (
	yyrpc "RPC"
	"RPC/xclient"
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"time"
)

type Foo int

//写成小写 什么都导不出去
type SumArgs struct {
	A int
	B int
}


//func (f *Foo) Sum(arg SumArgs,reply *int) error
//func (f *Foo) HelloWorld(args string,reply *string) error


func main() {
	log.SetLevel(log.InfoLevel)
	//conn,err := net.Dial("tcp",":9090")
	//if err != nil {
	//	log.Fatal("rpc client:dial error:",err)
	//}
	//client,err := yyrpc.NewClient(conn,&yyrpc.DefaultOption)

	//服务发现中心
	d := xclient.NewGeeRegistryDiscovery("http://localhost:9091/registry",time.Second*20)

	x := xclient.NewXClient(d,xclient.RoundRobinSelect,&yyrpc.DefaultOption)
	//毕竟是一个客户端，所以虽然提供多个服务的服务器仍然是同一种编码方式


	//if err != nil {
	//	log.Fatal("rpc client:newclient error:",err)
	//}
	ans := new(int)
	err := x.Call(context.Background(),"Foo.Sum",SumArgs{
		A: 1,
		B: 4,
	},ans)

	if err != nil {
		log.Info("rpc client:client call Foo.Sum error:",err)
	}
	fmt.Println("Ans:",*ans)


	ctx,_:= context.WithTimeout(context.Background(),time.Second*10)
	x.Call(ctx,"Foo.Sum",SumArgs{
		A: 1,
		B: 4,
	},ans)
	fmt.Println("Ans:",*ans)
}