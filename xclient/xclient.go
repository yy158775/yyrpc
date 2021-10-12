package xclient

import (
	"context"
	"io"
	"sync"
	"yyrpc"
)

//也是如果负载均衡了话，那么一个客户端，肯定只有一个option
type XClient struct {
	clients map[string]*yyrpc.Client
	mu sync.Mutex //锁
	d Discovery  //接口大小为16字节
	opt *yyrpc.Option
	mode SelectMode
}


//创建一个新的客户端，是一个注册中心还是什么
//如果创建多个客户端，肯定要公用一个注册中心

func NewXClient(d Discovery,mode SelectMode,opt *yyrpc.Option) *XClient {
	return &XClient{
		clients: make(map[string]*yyrpc.Client),
		d:       d,
		opt:     opt,
		mode:    mode,
	}
}

func (x *XClient) Close() error {
	//x.clients
	x.mu.Lock()
	defer x.mu.Unlock()
	for key,client := range x.clients {
		_ = client.Close()
		delete(x.clients,key)
	}
	return nil
}

var _ io.Closer = (*XClient)(nil)


//注册中心获得地址
func (x *XClient) Call(ctx context.Context,ServiceMethod string,argv,reply interface{}) error {
	rpcaddr,err := x.d.Get(x.mode)
	if err != nil {
		return err
	}
	return x.call(rpcaddr,ctx,ServiceMethod,argv,reply)
}

func (x *XClient) call(rpcaddr string,ctx context.Context,ServiceMethod string,argv,replyv interface{}) error {
	client,err := x.dial(rpcaddr)
	if err != nil {
		return err
	}
	return client.Call(ctx,ServiceMethod,argv,replyv)
}

func (x *XClient) dial(rpcaddr string) (client *yyrpc.Client,err error){
	x.mu.Lock() //一旦访问到clients
	defer x.mu.Unlock()

	client,ok := x.clients[rpcaddr]
	if ok && !client.IsAvailable() {
		_ = client.Close()
		delete(x.clients,rpcaddr)
		client = nil
	}
	if client == nil {
		var e error
		client,e = XDial(rpcaddr,x.opt)
		if e != nil {
			return nil,e
		}
		x.clients[rpcaddr] = client
	}
	return client,nil
}

func XDial(rpcaddr string,opt *yyrpc.Option) (*yyrpc.Client,error) {
	return yyrpc.Dial("tcp",rpcaddr,opt)
}