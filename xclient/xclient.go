package xclient

import (
	"RPC"
	"context"
	"io"
	log "github.com/sirupsen/logrus"
	"reflect"
	"sync"
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
	//fmt.Println("rpc xclient:dial:",rpcaddr)
	log.Debug("rpc xclient:dial ",rpcaddr)
	client,err := x.dial(rpcaddr)
	log.Debug("rpc xclient:dial return:",err)
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
		client,e = yyrpc.XDial(rpcaddr,x.opt)
		if e != nil {
			return nil,e
		}
		x.clients[rpcaddr] = client
	}
	return client,nil
}

func (xc *XClient) Broadcast(ctx context.Context, serviceMethod string, args, reply interface{}) error {
	serevrs,err := xc.d.GetAll()
	if err != nil {
		log.Println("rpc xclient:broadcast getall serves error:",err)
		return err
	}
	replydone := reply == nil
	ctx,cancel := context.WithCancel(ctx)

	var wg sync.WaitGroup
	var mu sync.Mutex

	for _,server := range serevrs {
		wg.Add(1)

		go func(rpcaddr string) {
			var e error
			defer wg.Done()
			var clonereply interface{} //定义接口时
			if !replydone {
				//clonereply = reflect.New(reflect.ValueOf(reply).Elem().Type()).Interface()
				clonereply = reflect.New(reflect.TypeOf(reply).Elem()).Interface()
			}

			//还是用的少，如果知道elem 是几种引用类型
			//Type才行，type类型的话，获取elem就是在获取它的元素类型。也不是指针吧，就是元素类型
			//Value 要求必须是指针或者接口
			e = xc.call(server,ctx,serviceMethod,args,clonereply)
			mu.Lock()
			if e != nil && err == nil {
				err = e
				cancel() //直接取消，然后函数就会返回吧 这里协程会结束
			}
			if err == nil && !replydone {
				reflect.ValueOf(reply).Elem().Set(reflect.ValueOf(clonereply).Elem())
				replydone = true
			}
			mu.Unlock()
		}(server)
	}
	wg.Wait()
	return err
}