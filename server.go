package yyrpc

import (
	"RPC/codec"
	//log "github.com/sirupsen/logrus"
	reg "RPC/registry"
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"net"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"time"
)

//服务端是要针对所有链接上这个服务器的端口
//进行解析，解码器是一个端口一个，注意依赖关系的不同
//服务器就是对每个接口创建一个解码器
const MagicNumber = 0x3bef5c

//const invalidRequest = "invalidRequest"

var invalidRequest = struct{}{}

type Option struct {
	MagicNumber int
	CodecType string
	ConnectTimeout time.Duration
	HandleTimeout time.Duration
}

type Server struct {
	serviceMap sync.Map //这里用这个数据结构的目的是什么
}

func NewServer() *Server {
	return &Server{}
}

var DefaultOption = Option{
	MagicNumber: MagicNumber,
	CodecType:   codec.GobType,
}
var DefaultServer = &Server{}


func (s *Server) Accept(lis net.Listener) {
	for	{
		conn,err := lis.Accept()
		if err != nil {
			log.Println("rpc server: accept error:",err) //log.Fatal
			return //直接返回 不是continue
		}
		go s.ServeConn(conn)
	}
}

func Accept(lis net.Listener) {
	DefaultServer.Accept(lis)
}

func (s *Server) ServeConn(conn io.ReadWriteCloser) {

	defer conn.Close()

	dec := json.NewDecoder(conn)
	var opt = &Option{}
	if err := dec.Decode(opt);err != nil {
		log.Println("rpc server: option decode error:",err)
		return
	}
	if opt.MagicNumber != MagicNumber {
		log.Println("rpc server: invalid MagicNumber")
		return
	}
	var f codec.NewCodecFunc
	if f = codec.NewCodecFuncMap[opt.CodecType];f == nil {
		log.Println("rpc server: invalid CodecType")
		return
	}
	s.ServeCodec(f(conn),opt) //并行了
}



func (s *Server) ServeCodec(cod codec.Codec,opt *Option) {
	sending := new(sync.Mutex)
	wg := new(sync.WaitGroup) //管定义一个指针肯定不行，要分配内存啊 顶一个指针加内存

	for {
		req,err := s.readRequest(cod)
		log.Debug("rpc server:request ",req)
		if err != nil {
			if req == nil {
				break //不可能恢复了 连头都没有读出来 结束这个链接  仔细分析一下错误的类型，计网各种错误的严重程度
			}
				req.h.Error = err.Error()
				s.sendResponse(sending, req.h, invalidRequest, cod) //至少发回一个头部，可以保存错误信息
				continue
		}
		wg.Add(1)
		go s.handleRequest(req,cod,wg,sending,opt.HandleTimeout)
	}
	wg.Wait() //等待欧所有携程完在结束
	 _ = cod.Close()
}

type request struct {
	h *codec.Header
	argv,replyv reflect.Value
	mtype *methodType //方法还需要什么
	svc *service //需要作为第一个参数么
}

func (s *Server) sendResponse(sending *sync.Mutex,h *codec.Header,body interface{},cod codec.Codec) {
	log.Debug("rpc server:sendResponse begin")
	sending.Lock()
	log.Debug("rpc server:sendResponse lock")
	defer sending.Unlock()
	if err := cod.Write(h,body);err != nil {  //程序通过defer操作已经将这个接口关闭了 在这关闭也可以
		//但是有设计到代码重用的方式
		log.Println("rpc server: write response err:",err)
	}
	log.Debug("rpc server:sendResponse end")
}

func (s *Server) handleRequest(req *request,cod codec.Codec,wg *sync.WaitGroup,sending *sync.Mutex,
								timeout time.Duration) {
	defer wg.Done()
	//req.argv
	//log.Println("header:",req.h," body:",req.argv.Elem())  //elem指针  这个value必须是指针
	//Kind is not Interface or Ptr 类型是接口
	//打印全部内容
	call := make(chan struct{})
	sent := make(chan struct{})
	go func() {
		err := req.svc.Call(req.mtype,req.argv,req.replyv)
		call<- struct{}{}
		if err != nil {
			req.h.Error = err.Error()
			s.sendResponse(sending, req.h, invalidRequest, cod)
			return
		}
		//req.replyv = reflect.ValueOf(fmt.Sprintf("geerpc resp %d",req.h.Seq))
		s.sendResponse(sending,req.h,req.replyv.Interface(),cod) //错误已经在这里处理过了
		//还是用的不熟练啊，这里不理解你后面的reflect.value 怎么可能理解
		sent<- struct{}{}
	}()

	if timeout == 0 {
		log.Debug("timeout:",timeout)
		<-call
		<-sent
		return
	}

	//之前没加 直接0s处理
	//rpc server:request handle timeout: except within 0s
	select {
	case <-call:
		<-sent
	case <-time.After(timeout): //立即返回一个通道，然后过一会这个通道会给你发消息
		req.h.Error = fmt.Sprintf("rpc server:request handle timeout: except within %s",timeout)
		s.sendResponse(sending,req.h,req.replyv.Interface(),cod)
	}
}

func (s *Server) readRequest(cod codec.Codec) (req *request,err error) {

	h,err := s.readRequestHeader(cod)  //为什么要把读headere单独封装起来，感觉没必要 因为头部是一个比较重要的地方，如果后面
	//出错了 好好思考如何设计协议太重要了这个
	if err != nil {
		fmt.Println("rpc server:readRequestHeader error:",err)
		return nil,err //直接返回nil，不用申请指针了 不可能恢复成功了
	}
	req = &request{h:h}

	svc,mtype,err := s.findService(h.ServiceMethod)  //打印过了已经
	if err != nil {
		return req,err
	}
	req.svc = svc
	req.mtype = mtype
	//req.argv = reflect.New(reflect.TypeOf(""))  //reflect.Value 类型和值 现申请了一个Value然后传入禁区
	//argv是指针类型没问题
	//reflect.ValueOf() 到底value里面是指针还是其他的值，不一定 看inerface吧
	req.argv = mtype.newArgv()
	req.replyv = mtype.newReply()
	//只要有那个函数即可
	//读取参数服务端，确保是一个指针？？？，但是传输了时候不是指针

	//反射得多思考思考
	//req.argv.K 这两个区别在哪里
	argvi := req.argv.Interface()
	if req.argv.Kind() != reflect.Ptr {
		argvi = req.argv.Addr().Interface() //就是求出接口对应的指针接口
	}
	if err := cod.ReadBody(argvi);err != nil {

	}
	return req,err
}

//为什么要单独包装
//error : 返回错误类型
//反正到外面都要关闭
func (s *Server) readRequestHeader(cod codec.Codec) (*codec.Header,error){
	h := &codec.Header{}
	if err := cod.ReadHeader(h);err != nil {  //在这里处理一下错误，然后将错误向上传递
		if err != io.EOF && err != io.ErrUnexpectedEOF {  //io错误的分类可以总结一下
			log.Println("rpc server:read header error:",err)
		}
		return nil,err
	}
	return h,nil
}

func (s *Server) Register(rcvr interface{}) error {  //为啥不注册一个类型呢？？，奇怪可能要作为第一个变量传入进去吧。
	//就是你实现这个服务，里面可以带有一定的，自带数据结构，怎么理解类和实例的关系，这里好好想一想
	service := newService(rcvr)
	if _,dup := s.serviceMap.LoadOrStore(service.name,service);dup {
		return errors.New("rpc server:already defined:"+service.name)
	}
	return nil
}

func Register(rcvr interface{}) error {
	return DefaultServer.Register(rcvr)
}


func (s *Server) findService(serviceMethod string) (svc *service,mtype *methodType,err error)  {
	dot := strings.LastIndex(serviceMethod,".")
	if dot < 0 {
		err = errors.New("rpc server: service.Method request ill-formed"+serviceMethod)
		return
	}
	servicename,methodname := serviceMethod[:dot],serviceMethod[dot+1:]
	svci,ok := s.serviceMap.Load(servicename) //svic
	if !ok {
		err = errors.New("rpc serevr:invalid service: " + servicename)
		return
	}
	svc = svci.(*service)
	mtype = svc.method[methodname] //两个map，一个是service，一个是method
	if mtype == nil {
		err = errors.New("rpc server:invalid method"+ methodname)
	}
	return
}


const (
	connected = "200 Connected to yy RPC"
	defaultRPCPath = "/_yyrpc_"
	defaultDebugPath = "/debug/yyrpc"
)

func (s *Server) ServeHTTP(writer http.ResponseWriter, r *http.Request) {
	if r.Method != "CONNECT" {
		//了解一下吧
		writer.Header().Set("Content-Type","text/plain;charset=utf-8")
		writer.WriteHeader(http.StatusMethodNotAllowed)
		_,_ = io.WriteString(writer,"405 must CONNECT\n")
		return
	}
}


func (s *Server) HandleHTTP() {
	http.Handle(defaultRPCPath,s)  //都是传入某个具体的实例，如果传入结构体 or 传入函数,面向对象的特性，不是面向过程
}

func HandleHTTP() {
	DefaultServer.HandleHTTP()
}


//要实现服务的那一段，隔一段时间发一个这个
func Heartbeat(registry,addr string,duration time.Duration) error {
	if duration == 0 {
		duration = reg.DefaultTimeout - time.Second
	}
	var err error //这就是共享变量吗？？ 没试过用一用试试吧
	err = sendHeartbeat(registry,addr)
	if err != nil {
		return err
	} else {
		go func() {
			ticker := time.NewTicker(duration) //这个可以思考一下怎么用
			for err == nil {
				<-ticker.C
				err = sendHeartbeat(registry, addr)
			}
		}()
	}
	return nil
}

func sendHeartbeat(registry,addr string) error {
	log.Println(addr, "send heart beat to registry", registry)
	httpclient := &http.Client{}

	//如何复用http链接
	req,err := http.NewRequest("POST",registry,nil)

	if err != nil {
		log.Fatal("rpc server:sendHeartbeat error:",err)
	}

	req.Header.Set("X-Servers",addr)
	//http.DefaultClient.Do()

	if _,err := httpclient.Do(req);err != nil {
		return err
		log.Println("rpc server: heart beat error:", err)
	}
	return nil
}