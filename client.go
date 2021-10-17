package yyrpc

import (
	"RPC/codec"
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Client struct {
	seq uint64
	cc codec.Codec
	opt *Option
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
		return 0, ErrShutDown
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

func Dial (network,address string,opts ...*Option) (*Client,error) {
	return dialTimeout(NewClient,network,address,opts...)
	//client,err := NewClient(conn,opt)
	//if err != nil {
	//	return nil,err
	//}
	//return client,nil 到最后可以优化吗
}

//这里也要想一想怎么回事吧
func parseOption(opts ...*Option)(*Option,error) {
	var opt *Option
	if len(opts) == 0 || opts[0] == nil {  //自己验证一下这里的语法结构啊
		opt = &DefaultOption
		return opt,nil
	}
	if len(opts) > 1 {
		return nil,errors.New("The number of option is more than 1")
	}
	opt = opts[0]
	opt.MagicNumber = DefaultOption.MagicNumber
	if opt.CodecType == "" {
		opt.CodecType = DefaultOption.CodecType
	}
	return opt,nil
}

func NewClient(conn net.Conn,opt *Option) (*Client,error) {
	f := codec.NewCodecFuncMap[opt.CodecType]
	if f == nil {
		return nil,errors.New("Invalid CodecType")
	}

	if err := json.NewEncoder(conn).Encode(opt);err != nil {
		return nil,err
	} //发送option

	return NewClientCodec(f(conn),opt),nil
}

func NewClientCodec(cod codec.Codec,opt *Option) *Client {
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
func (c *Client) Go(servciemethod string,args,replys interface{},done chan *Call) *Call {
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

func (c *Client) Call(ctx context.Context,servicemethod string,args,replys interface{}) error {
	done := make(chan *Call,1)
	call := c.Go(servicemethod,args,replys,done)
	//好像是这样有另外一个协程在既是吗？？
	//明天再弄吧？？ 增加日志打印出输出
	select {
	case <-ctx.Done():
		c.removeCall(call.Seq)
		return fmt.Errorf("rpc client:Call failed : %s",ctx.Err().Error())
	case call := <-call.Done:
		return call.Error
	}
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

type newClientFunc func(conn net.Conn,opt *Option) (*Client,error)

func dialTimeout(f newClientFunc,network,address string,opts ...*Option) (client *Client,err error) {
	opt,err := parseOption(opts...)
	if err != nil {
		return nil,err
	}

	conn,err := net.DialTimeout(network,address,opt.ConnectTimeout) //我怎么感觉等了双份时间
	if err != nil {
		return nil,err
	}
	defer func(){
		if err != nil {
			conn.Close()
		}
	}()

	type clientResult struct {
		client *Client
		err error
	}

	ch := make(chan *clientResult)

	go func() {
		c,err := f(conn,opt)

		//这种情况就是不会堵塞
		select {
		case ch <- &clientResult{ client: c, err:err }:
		default:
			return
		}
	}()
	//不知道为啥单列处理 ???? 等于0表示一直等待应该是
	if opt.ConnectTimeout == 0 {
		result := <-ch
		return result.client,result.err
	}

	select {
	case <-time.After(opt.ConnectTimeout):
		return nil,fmt.Errorf("rpc dial:connect timeout:%s",opt.ConnectTimeout)
	case result := <-ch :  //这里为零
		return result.client,result.err
	}
}


// NewHTTPClient new a Client instance via HTTP as transport protocol
func NewHTTPClient(conn net.Conn, opt *Option) (*Client, error) {
	_, _ = io.WriteString(conn, fmt.Sprintf("CONNECT %s HTTP/1.0\n\n", defaultRPCPath))

	// Require successful HTTP response
	// before switching to RPC protocol.
	resp, err := http.ReadResponse(bufio.NewReader(conn), &http.Request{Method: "CONNECT"})
	if err == nil && resp.Status == connected {
		return NewClient(conn, opt)
	}
	if err == nil {
		err = errors.New("unexpected HTTP response: " + resp.Status)
	}
	return nil, err
}

// XDial calls different functions to connect to a RPC server
// according the first parameter rpcAddr.
// rpcAddr is a general format (protocol@addr) to represent a rpc server
// eg, http@10.0.0.1:7001, tcp@10.0.0.1:9999, unix@/tmp/geerpc.sock
func XDial(rpcAddr string, opts ...*Option) (*Client, error) {
	parts := strings.Split(rpcAddr, "@")
	if len(parts) != 2 {
		return nil, fmt.Errorf("rpc client err: wrong format '%s', expect protocol@addr", rpcAddr)
	}
	protocol, addr := parts[0], parts[1]
	switch protocol {
	case "http":
		return DialHTTP("tcp", addr, opts...)
	default:
		// tcp, unix or other transport protocol
		return Dial(protocol, addr, opts...)
	}
}


// DialHTTP connects to an HTTP RPC server at the specified network address
// listening on the default HTTP RPC path.
func DialHTTP(network, address string, opts ...*Option) (*Client, error) {
	return dialTimeout(NewHTTPClient, network, address, opts...)
}

