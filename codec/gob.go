package codec

import (
	"bufio"
	"encoding/gob"
	"io"
)

type GobCodec struct {
	conn io.ReadWriteCloser //读写
	writer *bufio.Writer
	dec *gob.Decoder //只是个结构体，conn的关闭不是依靠他
	enc *gob.Encoder
}

func (g *GobCodec) Close() error {
	return g.conn.Close()
}

func (g *GobCodec) ReadHeader(header *Header) error {
	//if err := g.dec.Decode(header);err != nil {  //这个函数这个地方也用的interface{} 可以看看底层结构
	//	return err //分析一下底层数据结构
	//}
	//return nil
	return g.dec.Decode(header)
}

func (g *GobCodec) ReadBody(body interface{}) error {
	//gob.
	return g.dec.Decode(body)
}

func (g *GobCodec) Write(header *Header, body interface{}) error { //返回值怎么解决
	var err error
	defer func() {
		_ = g.writer.Flush()
		if err != nil {
			g.Close()
		}
	}()  //这种我还没怎么见过的，就是判断放在后面 分析一下这样逻辑的必要性
	//感觉写到后面也可以啊但是你可能得不断重复这个代码吧 要在所有可能返回的地方都把这个代码写一遍

	if err = g.enc.Encode(header);err != nil {
		return err
	}
	if err = g.enc.Encode(body);err != nil {
		return err
	}
	return nil
}

func NewGobCodec (conn io.ReadWriteCloser) Codec {  //接口从来没有指针吧
	buf := bufio.NewWriter(conn)
	return &GobCodec{
		conn : conn, //接口赋值怎么做
		writer: buf,
		dec: gob.NewDecoder(conn),
		enc:gob.NewEncoder(buf), //怎么引用啊？？？ 把这个提出来 不能同时初始化
	}
}

//这里面直接读写，错误都不在这里面处理，直接返回上层
//什么时候错误不返回上层，上层对这个错误不怎么需要的情况下吧