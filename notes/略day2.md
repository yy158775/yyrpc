# 略day2

![image-20211009205102995](/home/yy/.config/Typora/typora-user-images/image-20211009205102995.png)

要求：

- method的tyep必须是暴露的
- method是可导出的
- 有两个参数
- 第二个参数是一个指针
- 返回一个error

```go
func (t *T) MethodName(argType T1, replyType *T2) error
```

封装call

```go
// Call represents an active RPC.
type Call struct {
	Seq           uint64
	ServiceMethod string      // format "<service>.<method>"
	Args          interface{} // arguments to the function
	Reply         interface{} // reply from the function
	Error         error       // if error occurs, it will be set
	Done          chan *Call  // Strobes when call is complete.
}

func (call *Call) done() {
	call.Done <- call
}
//如果结束了就向通道
//为了支持异步调用，Call 结构体中添加了一个字段 Done，Done 的类型是 chan *Call，当调用结束时，会调用 call.done() 通知调用方。
//当地调用结束时？？？
```

# 时序图

什么是时序图，序列图，UML交互图，

时序图的组成元素：

- 角色：actor   可以是人或者其他系统和子系统
- 对象：object  时序图的顶部，以一个举行表示
- 



# 支持异步和并发的高性能客户端

## 实现client

```go
// Client represents an RPC Client.
// There may be multiple outstanding Calls associated
// with a single Client, and a Client may be used by
// multiple goroutines simultaneously.

可能被几个协程同时使用有可能。 //这个问题要考虑清楚啊
//支持并发？？？	

type Client struct {
	cc       codec.Codec
	opt      *Option
	sending  sync.Mutex // protect following
	header   codec.Header
	mu       sync.Mutex // protect following
	seq      uint64
	pending  map[uint64]*Call
	closing  bool // user has called Close
	shutdown bool // server has told us to stop
}

var _ io.Closer = (*Client)(nil)

var ErrShutdown = errors.New("connection is shut down")

// Close the connection
func (client *Client) Close() error {
	client.mu.Lock()
	defer client.mu.Unlock()
	if client.closing {
		return ErrShutdown   //到底是谁关闭的吗？？
	}
	client.closing = true
	return client.cc.Close()
}

// IsAvailable return true if the client does work
func (client *Client) IsAvailable() bool {
	client.mu.Lock()
	defer client.mu.Unlock()
	return !client.shutdown && !client.closing
}
```

​	pending 存储未处理完的请求，键是编号，值是 Call 实例。

​	seq 用于给发送的请求编号，每个请求拥有唯一编号。

​	header 是每个请求的消息头，header 只有在请求发送时才需要，而请求发送是互斥的，因此每个客户端只需要一个，声明在 Client 结构体中可以复用。









```
return c.cc.Close() //能有啥错误呢？？？
```

![image-20211010121514171](/home/yy/.config/Typora/typora-user-images/image-20211010121514171.png)

![image-20211010134543858](/home/yy/.config/Typora/typora-user-images/image-20211010134543858.png)	



# 项目

```go
if err != nil {
	call.Error = err
	call.done()
	return
}
```

```go
args := fmt.Sprintf("geerpc req %d", i)
var reply string

if err := client.Call("Foo.Sum", args, &reply); err != nil {
	log.Fatal("call Foo.Sum error:", err)
}


// Call invokes the named function, waits for it to complete,
// and returns its error status.
func (client *Client) Call(serviceMethod string, args, reply interface{}) error {
	call := <-client.Go(serviceMethod, args, reply, make(chan *Call, 1)).Done
    //make(chan *Call, 1) 使用go，什么时候返回呢？？？
	return call.Error
}



// Go invokes the function asynchronously.
// It returns the Call structure representing the invocation.
func (client *Client) Go(serviceMethod string, args, reply interface{}, done chan *Call) *Call {
	if done == nil {
		done = make(chan *Call, 10)
	} else if cap(done) == 0 {
		log.Panic("rpc client: done channel is unbuffered")
	}
	call := &Call{
		ServiceMethod: serviceMethod,
		Args:          args,
		Reply:         reply,
		Done:          done,
	}
	client.send(call)
	return call
}
```

Go完之后，里面函数进行send，然后就是，等待了吧。



```go
func main() {
    log.SetFlags(0)
	addr := make(chan string)
	
    //开启服务器
    go startServer(addr)
    
    //geerpc.Dial()
    
	client, _ := geerpc.Dial("tcp", <-addr)
	defer func() { _ = client.Close() }()

    
	time.Sleep(time.Second)
	// send request & receive response
	var wg sync.WaitGroup
	
    for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			args := fmt.Sprintf("geerpc req %d", i)
			var reply string
			if err := client.Call("Foo.Sum", args, &reply); err != nil {
				log.Fatal("call Foo.Sum error:", err)
			}
			log.Println("reply:", reply)
		}(i)
	}
    
	wg.Wait()
}
```



```go
// Dial connects to an RPC server at the specified network address
func Dial(network, address string, opts ...*Option) (client *Client, err error) {
	opt, err := parseOptions(opts...)
	if err != nil {
		return nil, err
	}
	conn, err := net.Dial(network, address)
	if err != nil {
		return nil, err
	}
	// close the connection if client is nil
	defer func() {
		if client == nil {
			_ = conn.Close()
		}
	}()
	return NewClient(conn, opt)
}
```





# 项目细节总结

```go
	if client.closing {
		return ErrShutdown
	} 
//如果已经关闭了话，就返回ErrShutdown
//closing 是用户主动关闭的，即调用 Close 方法，而 shutdown 置为 true 一般是有错误发生。
//header 是每个请求的消息头，header 只有在请求发送时才需要，而请求发送是互斥的，因此每个客户端只需要一个，声明在 Client 结构体中可以复用。
//为了复用啊，真聪明 但是error可能不同，属于返回值的一部分还要使用，后面继续判断

```

为了复用，发送时候和接受时候都是互斥，那么可以都只用一个就行了吗？？接受时候不怕被更改吗？

```go
registerCall
removeCall
terminateCalls //所有函数调用全部返回
//这个修改了shutdown
```

接受功能：

却是你要仔细想想，接受到call后发现不存在有哪些可能：

```go
    call 不存在，可能是请求没有发送完整，或者因为其他原因被取消，但是服务端仍旧处理了。
//没有发送完整：只发送了头
//被取消了，发送完好，被别人取消了
    call 存在，但服务端处理出错，即 h.Error 不为空。
//服务端处理出错，返回错误了，就是说返回值不可用了
    call 存在，服务端处理正常，那么需要从 body 中读取 Reply 的值。
//正常情况
```

仔细思考一下：

```go
ReadBody
ReadHead
Write
//这些都有可能出错 在recevie的时候，这些都有可能出错的
```

```go
removeCall的位置：
	1、接收到请求时： 接收到的要remove走
	2、发送请求失败时：但是此时有可能 remove为nil，原因是：可能头写入了，但是参数未写入吧。但是如果两个都没写入，那么收不到请求，还是在这里移走吧
	此时应为发送失败了，在等待相应也没什么意思了吧？？
```



```
	Go 异步的，发送完毕后，不会等待结束。得到的参数也不一定可用，但是不同。Go可以控制 receive 会调用Done
Call
```

```go
call.done()
	receive:接受请求时
	send：本身就失败了时候
```

返回值call.Error

```go
Go(serviceMethod string, args, reply interface{}, done chan *Call) *Call
//注意reply是外部提供的指针类型吧。这样reply直接由外部提供

Call(serviceMethod string, args, reply interface{}) error

func (client *Client) send(call *Call)
```

