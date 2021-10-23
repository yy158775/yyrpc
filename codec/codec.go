package codec

import "io"

type Header struct {
	ServiceMethod string
	//服务和方法
	Seq uint64
	Error string
	//返回的error
}

type Body interface {} //发送的参数
//返回的结果

type Codec interface {
	io.Closer
	ReadHeader(header *Header) error
	ReadBody(body interface{}) error
	Write(header *Header,body interface{}) error
}


const (
	GobType = "GobType"
	JsonType = "JsonType"
)

type NewCodecFunc func(closer io.ReadWriteCloser) Codec

var NewCodecFuncMap map[string]NewCodecFunc

func init() {
	NewCodecFuncMap = make(map[string]NewCodecFunc)
	NewCodecFuncMap[GobType] = NewGobCodec
	//NewCodecFuncMap[JsonType] = NewJsonCodec
}