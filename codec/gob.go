package codec

import (
	"bufio"
	"encoding/gob"
	"io"
)

type GobCodec struct {
	conn io.ReadWriteCloser
	writer *bufio.Writer
	dec *gob.Decoder
	enc *gob.Encoder
}

func (g *GobCodec) Close() error {
	return g.conn.Close()
}

func (g *GobCodec) ReadHeader(header *Header) error {
	return g.dec.Decode(header)
}

func (g *GobCodec) ReadBody(body interface{}) error {
	return g.dec.Decode(body)
}

func (g *GobCodec) Write(header *Header, body interface{}) error {
	var err error
	defer func() {
		_ = g.writer.Flush()
		if err != nil {
			g.Close()
		}
	}()

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
		conn : conn,
		writer: buf,
		dec: gob.NewDecoder(conn),
		enc:gob.NewEncoder(buf),
	}
}
