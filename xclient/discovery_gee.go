package xclient

import (
	"log"
	"net/http"
	"strings"
	"time"
)

type GeeRegisteryDiscovery struct {
	*MultiServerDiscovery
	registry string  //
	timeout time.Duration
	lastUpdate time.Time
}

func (g GeeRegisteryDiscovery) Refresh() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.lastUpdate.Add(g.timeout).After(time.Now()) {
		return nil
	}
	log.Println("rpc dicovery:refresh from registry:",g.registry)
	resp,err := http.Get(g.registry)

	if err != nil {
		log.Println("rpc discovery:http get error:",err)
		return err
	}

	servers := resp.Header.Get("X-Servers")

	g.servers =	strings.Split(servers,",")

	return nil
}

func (g GeeRegisteryDiscovery) Update(servers []string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.lastUpdate = time.Now()
	g.servers = servers
	return nil
}

func (g GeeRegisteryDiscovery) Get(mode SelectMode) (string, error) {
	//为啥要先刷新，这个过期时间到底怎么设置的。
	if err := g.Refresh();err != nil {
		return "", err
	}
	return g.MultiServerDiscovery.Get(mode)
}

func (g GeeRegisteryDiscovery) GetAll() ([]string, error) {
	if err := g.Refresh();err != nil {
		return nil, err
	}
	return g.MultiServerDiscovery.GetAll()
}

const defaultUpdateTimeout = time.Second * 10

func NewGeeRegistryDiscovery(registry string,timeout time.Duration) *GeeRegisteryDiscovery {
	if timeout == 0 {
		timeout = defaultUpdateTimeout //才10s 就要更新一次服务列表
	}

	g := &GeeRegisteryDiscovery{
		MultiServerDiscovery: NewMultiServerDiscovery(nil),
		registry:             registry,
		timeout:              timeout,
	}
	return g
}