package client

import (
	"encoding/json"
	"errors"
	"log"
	"net"
	"sync"
	server "yyrpc"
	"yyrpc/codec"
)

type Client struct {
	seq uint64
	cc codec.Codec
	opt *server.Option
	header codec.Header
	mu sync.RWMutex
	sending sync.Mutex
	pending map[uint64]*Call
	closing bool
	shutdown bool
}

type Call struct {
	Seq uint64
	Error error
	ServiceMethod string
	Args,reply interface{}
	Done chan *Call
}

func (c *Call) done() {
	c.Done <- c
}

var ErrShutDown = errors.New("connection is shut down")
//不能做常量吗？

func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closing {
		log.Println("rpc client:has already closed")
		return ErrShutDown
	}
	//if err := c.cc.Close();err != nil {
	//	log.Println("rpc client:close error:",err)
	//	return err
	//} 不用处理错误就是这么简单
	c.closing = true
	return c.cc.Close()
	//里面没有conn这个结构体
}

func (c *Client) IsAvailable() bool {
	return !c.shutdown && !c.closing
}

func (c *Client) registerCall(call *Call) (uint64,error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.IsAvailable() {
		return 0,ErrShutDown
	}
	call.Seq = c.seq
	c.seq++
	c.pending[call.Seq] = call
	return call.Seq,nil
}

func (c *Client) removeCall(seq uint64) *Call {
	c.sending.Lock()
	defer c.sending.Unlock()
	delete(c.pending, seq)
	return c.pending[seq]
}

func (c *Client) terminateCalls() {
	c.sending.Lock()
	defer c.sending.Unlock()
	c.mu.Lock()
	defer c.mu.Unlock()
	c.shutdown = true
	for _,call := range c.pending {
		call.done()
		delete(c.pending, call.Seq)
	}
}

func (c *Client) receive() {
	var h codec.Header
	var err error
	for err == nil {
		err = c.cc.ReadHeader(&h)
		if err != nil {
			break
		}
		call := c.pending[h.Seq]
		switch {
		case call == nil :
			c.cc.ReadBody(nil) //没有说明什么已经去除了，可能只写了个头部
		case h.Error != "" :  //string 没有 nil 不行吗
			call.Error = errors.New(h.Error)
			c.cc.ReadBody(nil) //只写了个头部但是还没来得及去除
			//或者其他问题
		default:
			err = c.cc.ReadBody(call.reply)
			if err != nil {
				call.Error = errors.New("rpc client:read body error:"+err.Error())
			}
			call.done()
		}
	}

	//c.Close()
	c.terminateCalls() //Close和这个有什么区别呢？？ 把等待列表里的全弄没了，不等待接受了，直接全部结束了
	//这个一般是服务端出错了，就是读取失败了，返回错误了shutdown
}

func Dial (network,address string,opts ...*server.Option) (*Client,error) {
	opt,err := parseOption(opts...)
	if err != nil {
		return nil,err
	}
	conn,err := net.Dial(network,address)
	if err != nil {
		return nil,err
	}
	defer func(){
		if err != nil {
			conn.Close()
		}
	}()

	//client,err := NewClient(conn,opt)
	//if err != nil {
	//	return nil,err
	//}
	//return client,nil 到最后可以优化吗
	return NewClient(conn,opt)
}

//这里也要想一想怎么回事吧
func parseOption(opts ...*server.Option)(*server.Option,error) {
	var opt *server.Option
	if len(opts) == 0 || opts[0] == nil {  //自己验证一下这里的语法结构啊
		opt = &server.DefaultOption
		return opt,nil
	}
	if len(opts) > 1 {
		return nil,errors.New("The number of option is more than 1")
	}
	opt = opts[0]
	opt.MagicNumber = server.DefaultOption.MagicNumber
	if opt.CodecType == "" {
		opt.CodecType = server.DefaultOption.CodecType
	}
	return opt,nil
}

func NewClient(conn net.Conn,opt *server.Option) (*Client,error) {
	f := codec.NewCodecFuncMap[opt.CodecType]
	if f == nil {
		return nil,errors.New("Invalid CodecType")
	}

	if err := json.NewEncoder(conn).Encode(opt);err != nil {
		return nil,err
	} //发送option

	return NewClientCodec(f(conn),opt),nil
}

func NewClientCodec(cod codec.Codec,opt *server.Option) (*Client) {
	client := &Client{
		seq:	  1,
		cc:       cod,
		opt:      opt,
		pending:  make(map[uint64]*Call),
		closing:  false,
		shutdown: false,
	}
	go client.receive()
	return client //在这里启动即可
}

//replys 必须是指针类型
func (c *Client) Go(servciemethod string,args,replys interface{},done chan *Call) *Call{
	//排除异常情况啊哥哥
	if done == nil {  //len(done) 里面如果是空的怎么办
		done = make(chan *Call,10) //默认使用什么吧
	} else if cap(done) == 0 { //是不是可以使用剩余的cap() ????
		log.Panic("rpc client: done channel is unbuffered") //设置的不好吧
	}
	call := &Call{
		ServiceMethod: servciemethod,
		Args:          args,
		reply:         replys,
		Done:          done,
	}
	go c.send(call) //可以通过同步的方法
	return call
}

func (c *Client) Call(servicemethod string,args,replys interface{}) error {
	done := make(chan *Call,1)
	call := c.Go(servicemethod,args,replys,done)
	<-call.Done
	return call.Error
}

func (c *Client) send(call *Call) {
	c.sending.Lock()
	defer c.sending.Unlock()

	seq,err := c.registerCall(call) //注册都失败了
	if err != nil {
		call.done()
		call.Error = err
		return
	}

	//c.mu.Lock()其他地方的代码不会改这个位置
	c.header.Seq = seq
	c.header.ServiceMethod = call.ServiceMethod
	c.header.Error = ""
	//c.mu.Unlock()

	if err := c.cc.Write(&c.header,call.Args);err != nil {
		call := c.removeCall(call.Seq)
		if call != nil {  //已经被接受处理了吧
			call.done()
			call.Error = err  //不一定是服务端来的error，而是客户端自身产生的错误
		}
	}
}

