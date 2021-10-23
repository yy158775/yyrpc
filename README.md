# 项目简介

简单的RPC框架

支持超时处理，负载均衡，异步和并发调用函数

RPC基于TCP协议，采用go内置gob编码解码器

# 超时处理

三种超时机制处理

- 和服务端建立连接时超时处理
- 调用函数时，写请求，服务端处理请求，读相应总时间的超时处理
- 服务端处理请求的超时处理

# 负载均衡

​	服务发现更新timeout：const defaultUpdateTimeout = time.Second * 10 

​	如果10s内更新过，就不用更新了，否则直接获取

​	注册中心：5分钟过期一次

```go
DefaultTimeout = time.Minute * 5
```

服务端注册：发送http请求，post请求

服务发现：发送http get请求获得服务器地址

## 负载均衡策略

- 随机选择策略

- 轮训算法

# 项目结构

├── client.go
├── client_test.go
├── codec
│   ├── codec.go
│   └── gob.go
├── example
│   ├── client
│   │   ├── client
│   │   └── client.go
│   ├── main
│   │   └── main.go
│   ├── register
│   │   └── registry.go
│   └── server
│       └── server.go
├── go.mod
├── go.sum
├── registry
│   └── registry.go
├── server.go
├── service.go
└── xclient
    ├── discovery_gee.go
    ├── discovery.go
    └── xclient.go

# example

client

register

server

三个目录

一起运行即可
