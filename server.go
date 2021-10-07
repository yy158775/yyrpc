package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"yyrpc/codec"
)

//服务端是要针对所有链接上这个服务器的端口
//进行解析，解码器是一个端口一个，注意依赖关系的不同
//服务器就是对每个接口创建一个解码器
const MagicNumber = 0x3bef5c

type Option struct {
	MagicNumber int
	CodecType string
}

type Server struct {

}

var DefaultOption = Option{
	MagicNumber: MagicNumber,
	CodecType:   codec.GobType,
}
var DefaultServer = &Server{}

//这个项目里面仔细分析一下，数据结构各个什么时候关闭，什么时候坑需要关闭，仔细想一下
//每个数据结构的存在周期

func (s *Server) Accept(lis net.Listener) {
	for conn,err := lis.Accept() {
		if err != nil {

		}
		s.ServeConn(conn)
	}
}

func Accept(lis net.Listener) {
	for conn := lis.Accept() {
		DefaultServer.ServeConn(conn)
	}
}

func (s *Server) ServeConn(conn io.ReadWriteCloser) {
	dec := json.NewDecoder(conn)
	var opt = &Option{}
	if err := dec.Decode(opt);err != nil {
		//return nil
		fmt.Println()
	}
	if opt.MagicNumber != {

	}
	//if opt.CodecType =
	var f codec.NewCodecFunc
	if f = codec.NewCodecFuncMap[opt.CodecType];f == nil {
		fmt.Println()
	}
	s.ServeCodec(f(conn))
}

func (s *Server) ServeCodec(cod codec.Codec) {
	for {
		var opt codec.Header
		if err := cod.ReadHeader(&opt);err != nil {

		}
		var body codec.Body  //如何解析参数我很迷惑 函数名不是用户使用的时候定义吗
		if err := cod.ReadBody(body);err != nil {

		}

	}
}