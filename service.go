package yyrpc

import (
	"go/ast"
	"log"
	"reflect"
	"sync/atomic"
)

type methodType struct {
	method reflect.Method
	ArgType reflect.Type
	ReplyType reflect.Type
	numCalls uint64
}

func (m *methodType) NumCalls() uint64 {
	return m.numCalls
}
//返回 如何由type得到value类型的
//为什么这个不用给开辟空间，我也不清楚
func (m *methodType) newArgv() reflect.Value {
	var argv reflect.Value
	if m.ArgType.Kind() == reflect.Ptr {
		argv = reflect.New(m.ArgType.Elem())
	} else {
		argv = reflect.New(m.ArgType).Elem()  //new和valueof
	}
	return argv
}

//返回值必须是指针
//func (t *T) MethodName(argType T1, replyType *T2) error
//定义是指针
func (m *methodType) newReply() reflect.Value {
	//if m.ReplyType. 这里应该是最难的了 返回值如何创建，如何写入进去，必须是指针吧
	//map slice还是双层结构
	var replyv reflect.Value
	replyv = reflect.New(m.ReplyType.Elem())
	//如果是指针的话，这个里面放的是地址啊
	//if reflect.Indirect(replyv).Kind() == reflect.Slice {
	//	replyv.Set(reflect.MakeSlice(m.ReplyType,0,1))
	//} else if reflect.Indirect(replyv).Kind() == reflect.Map{
	//	replyv.Set(reflect.MakeMap(m.ReplyType))
	//}
	switch replyv.Elem().Kind() {
	case reflect.Map:
		replyv.Elem().Set(reflect.MakeMap(m.ReplyType))
	case reflect.Slice:
		replyv.Elem().Set(reflect.MakeSlice(m.ReplyType,0,0))
	}
	return replyv
}

type service struct {
	name string
	typ reflect.Type
	rcvr reflect.Value  //reciver
	method map[string]*methodType
}

func newService(rcvr interface{}) *service {
	s := new(service)

	s.rcvr = reflect.ValueOf(rcvr)

	s.name = reflect.Indirect(s.rcvr).Type().Name() //不确定s.rcvr是不是指针吧

	s.typ = reflect.TypeOf(rcvr)

	if !ast.IsExported(s.name) {
		log.Fatalf("rpc server: %s is not a valid service name",s.name)
	}
	s.registerMethods()
	return s
}

func (s *service) registerMethods() { //给某个具体的service进行注册
	s.method = make(map[string]*methodType)
	for i := 0;i < s.typ.NumMethod();i++ {
		method := s.typ.Method(i)
		mtype := method.Type
		if mtype.NumIn() != 3 || mtype.NumOut() != 1 {
			continue
		}
		//如何获得一个Type是一个接口类型,先用接口的指针，再用elem
		//函数的返回值是接口的情况
		if mtype.Out(0) != reflect.TypeOf((*error)(nil)).Elem() {

		}
		argType,replyType := mtype.In(1),mtype.In(2)
		if !isExportedOrBuiltinType(argType) || !isExportedOrBuiltinType(replyType) {
			continue
		}
		s.method[method.Name] = &methodType{
			method:    method,
			ArgType:   argType,
			ReplyType: replyType,
			numCalls:  0,
		}
		log.Printf("rpc server: register %s:%s\n",s.name,method.Name)
	}
}

func isExportedOrBuiltinType(t reflect.Type) bool {
	return ast.IsExported(t.Name()) || t.PkgPath() == ""
}

func (s *service) Call(m *methodType,argv,replyv reflect.Value) error {
	atomic.AddUint64(&m.numCalls,1) //注意并发

	f := m.method.Func

	returnValues := f.Call([]reflect.Value{s.rcvr,argv,replyv}) //获取它的接口，接口为nil，这里需要要理解一下

	if errInter := returnValues[0].Interface();errInter != nil { //获取它的接口，接口为nil，这里需要要理解一下
		return errInter.(error)
	}
	return nil
}

