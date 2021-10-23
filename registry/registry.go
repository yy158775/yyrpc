package registry

import (
	"io"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

type YyRegistry struct {
	timeout time.Duration
	mu sync.Mutex
	servers map[string]*ServerItem
}

type ServerItem struct {
	addr string
	start time.Time
}

const (
	defaultPath    = "/_geerpc_/registry"
	DefaultTimeout = time.Minute * 5
)

func New(timeout time.Duration) *YyRegistry {
	return &YyRegistry{
		timeout: timeout,
		servers: make(map[string]*ServerItem),
	}
}

var DefaultYyRegister = New(DefaultTimeout)

func (r *YyRegistry) putServer(addr string) {
	if addr == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	item := r.servers[addr]
	if item == nil {
		r.servers[addr] = &ServerItem{
			addr:  addr,
			start: time.Now(),
		}
	} else {
		r.servers[addr].start = time.Now()
	}
}


func (r *YyRegistry) aliveServers() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	servers := make([]string,0)
	for addr,server := range r.servers {
		if r.timeout == 0 || server.start.Add(r.timeout).After(time.Now()) {
			servers = append(servers,addr)
		} else {
			delete(r.servers,addr)
		}
	}
	sort.Strings(servers)
	return servers
}

func (r *YyRegistry) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	switch request.Method {
	case "GET":
		writer.Header().Set("X-Servers",strings.Join(r.aliveServers(),","))
	case "POST":
		addr := request.Header.Get("X-Servers")
		if addr == "" {
			io.WriteString(writer,"Post Method Need X-Servers\n")
			return
		}
		r.putServer(addr)  //错误处理
		//会返回请求吗？？怎么no code
	default:
		io.WriteString(writer,"Invalid HTTP Method\n")
	}
}


func (r *YyRegistry) HandleHTTP(registryPath string) {
	http.Handle(registryPath,r)
	log.Println("rpc registry path:",registryPath)
}

func HandleHTTP() {
	DefaultYyRegister.HandleHTTP(defaultPath)
}
