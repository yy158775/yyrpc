package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"reflect"
	"sync"
	"yyrpc/codec"
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
}

type Server struct {

}

func NewServer() *Server {
	return &Server{}
}

var DefaultOption = Option{
	MagicNumber: MagicNumber,
	CodecType:   codec.GobType,
}
var DefaultServer = &Server{}

//这个项目里面仔细分析一下，数据结构各个什么时候关闭，什么时候坑需要关闭，仔细想一下
//每个数据结构的存在周期

func (s *Server) Accept(lis net.Listener) {
	for	{
		conn,err := lis.Accept()
		if err != nil {
			log.Println("rpc server: accept error:",err) //log.Fatal
			return //直接返回 不是continue
		}
		s.ServeConn(conn)
	}
}

func Accept(lis net.Listener) {
	DefaultServer.Accept(lis) //包装一下
}

func (s *Server) ServeConn(conn io.ReadWriteCloser) {

	defer conn.Close()  //在这关闭 如果之前已经退出关闭过了会不会有影响

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
	s.ServeCodec(f(conn))
}



func (s *Server) ServeCodec(cod codec.Codec) {
	//sending := sync.Mutex{}
	sending := new(sync.Mutex)
	wg := new(sync.WaitGroup) //管定义一个指针肯定不行，要分配内存啊 顶一个指针加内存

	for {
		req,err := s.readRequest(cod)
		if err != nil {
			if req == nil {
				fmt.Println("break")
				break //不可能恢复了 连头都没有读出来 结束这个链接  仔细分析一下错误的类型，计网各种错误的严重程度
			}
				fmt.Println("after break")
				req.h.Error = err.Error()
				s.sendResponse(sending, req.h, invalidRequest, cod) //至少发回一个头部，可以保存错误信息
				continue
		}
		wg.Add(1)
		go s.handleRequest(req,cod,wg,sending)
	}
	wg.Wait() //等待欧所有携程完在结束
	 _ = cod.Close()
}

type request struct {
	h *codec.Header
	argv,replyv reflect.Value
}

func (s *Server) sendResponse(sending *sync.Mutex,h *codec.Header,body interface{},cod codec.Codec) {
	sending.Lock()
	defer sending.Unlock()
	if err := cod.Write(h,body);err != nil {  //程序通过defer操作已经将这个接口关闭了
		log.Println("rpc server: write response err:",err)
	}
}

func (s *Server) handleRequest(req *request,cod codec.Codec,wg *sync.WaitGroup,sending *sync.Mutex) {
	defer wg.Done()
	//req.argv
	log.Println(req.h,req.argv.Elem())  //elem指针  这个value必须是指针
	//Kind is not Interface or Ptr 类型是接口
	//打印全部内容
	req.replyv = reflect.ValueOf(fmt.Sprintf("geerpc resp %d",req.h.Seq))
	s.sendResponse(sending,req.h,req.replyv.Interface(),cod) //错误已经在这里处理过了
	//还是用的不熟练啊，这里不理解你后面的reflect.value 怎么可能理解
}

func (s *Server) readRequest(cod codec.Codec) (req *request,err error) {
	h,err := s.readRequestHeader(cod)  //为什么要把读headere单独封装起来，感觉没必要 因为头部是一个比较重要的地方，如果后面
	//出错了 好好思考如何设计协议太重要了这个
	if err != nil {
		fmt.Println("readRequest error:",err)
		return nil,err //直接返回nil，不用申请指针了 不可能恢复成功了
	}
	req = &request{h:h}

	req.argv = reflect.New(reflect.TypeOf(""))  //reflect.Value 类型和值 现申请了一个Value然后传入禁区
	//argv是指针类型没问题
	//reflect.ValueOf() 到底value里面是指针还是其他的值，不一定 看inerface吧
	if err = cod.ReadBody(req.argv.Interface());err != nil {
		log.Println("rpc server:read body error:",err) //这里不直接返回，那怎么办，错误信息没有了吗
	}
	log.Println("Body:",req.argv.Elem())
	return req,err
}

//为什么要单独包装
func (s *Server) readRequestHeader(cod codec.Codec) (*codec.Header,error){
	h := &codec.Header{}
	if err := cod.ReadHeader(h);err != nil {
		if err != io.EOF && err != io.ErrUnexpectedEOF {  //io错误的分类可以总结一下
			log.Println("rpc server:read header error:",err)
		}
		return nil,err
	}
	return h,nil
}