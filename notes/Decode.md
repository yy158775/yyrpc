# Decode

// Decode reads the next value from the input stream and stores
// it in the data represented by the empty interface value.

从流中读取下一个value，将他存在这个空的interface value中。

// If e is nil, the value will be discarded. Otherwise,
// the value underlying e must be a pointer to the
// correct type for the next data item received.
// If the input is at EOF, Decode returns io.EOF and
// does not modify e.

这个value底层必须是一个指针指向正确的类型下一个接收到的item。

如果输入是EOF，解码返回io.EOF

```go
if e == nil {
   return dec.DecodeValue(reflect.Value{})
}
value := reflect.ValueOf(e)
if value.Type().Kind() != reflect.Ptr {
    dec.err = errors.New("gob: attempt to decode into a non-pointer")
    return dec.err
}
	return dec.DecodeValue(value)
```



// decodeTypeSequence parses:
// TypeSequence
// (TypeDefinition DelimitedTypeDefinition*)?
// and returns the type id of the next value. It returns -1 at
// EOF.  Upon return, the remainder of dec.buf is the value to be
// decoded. If this is an interface value, it can be ignored by
// resetting that buffer.



// A typeId represents a gob Type as an integer that can be passed on the wire.
// Internally, typeIds are used as keys to a map to recover the underlying type info.

被用来作为key 去映射恢复底层的type info

type typeId int32

```go
id := dec.decodeTypeSequence(false)
if dec.err == nil {
   dec.decodeValue(id, v)
}
func (dec *Decoder) decodeValue(wireId typeId, value reflect.Value) {
    
}
```

![image-20211008003453262](/home/yy/.config/Typora/typora-user-images/image-20211008003453262.png)

如果是结构体那就不断递归下去，也很快

![image-20211008003739919](/home/yy/.config/Typora/typora-user-images/image-20211008003739919.png)

# 反写错误

```go
func (g *GobCodec) ReadHeader(header *Header) error {
   //if err := g.dec.Decode(header);err != nil {  //这个函数这个地方也用的interface{} 可以看看底层结构
   // return err //分析一下底层数据结构
   //}
   //return nil
   return g.dec.Decode(header)
}

错误传递了三个位置

	if err := cod.ReadHeader(h);err != nil {
		if err != io.EOF && err != io.ErrUnexpectedEOF {  //io错误的分类可以总结一下
			log.Println()
		}
		return nil,err
	}


	req,err := s.readRequest(cod)
	if err != nil {
		if req == nil {
			break //不可能恢复了 连头都没有读出来 结束这个链接  仔细分析一下错误的类型，计网各种错误的严重程度
			}
			req.h.Error = err.Error()
			s.sendResponse(sending, req.h, invaildRequest, cod) //至少发回一个头部，可以保存错误信息
				continue
		}

```

如果头部没错，就一定会给你一个错误的返回信息，如果头部都没读出来，说明链接出错了，直接断开连接。

如果是readbody是出错了呢，多问几个为什么吧？？

然后把错误发送回去，多想想弄出来个模型。