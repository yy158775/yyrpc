package codec

import "io"

type Header struct {
	ServiceMethod string //服务和方法
	Seq int64
	Error error //返回的error
}

type Body interface {} //发送的参数
//返回的结果

type Codec interface {
	io.Closer
	ReadHeader(header *Header) error
	ReadBody(body interface{}) error //使用接口？？ 不使用指针吧
	//一直到客户引用我的包的时候我才知道具体的数据结构是什么
	Write(header *Header,body interface{}) error
}


const (
	GobType = "GobType"
	JsonType = "JsonType"
)

type NewCodecFunc func(closer io.ReadWriteCloser) Codec  //接口有指针吗？

var NewCodecFuncMap map[string]NewCodecFunc

func init() {
	NewCodecFuncMap[GobType] = NewGobCodec
	NewCodecFuncMap[JsonType] = NewJsonCodec
}