package xclient

import (
	"errors"
	"math"
	"math/rand"
	"sync"
	"time"
)

type SelectMode int


const (
	RoundRobinSelect SelectMode = iota
	RandomSelect
)

type Discovery interface {
	Refresh() error
	Update(servers []string) error
	Get(mode SelectMode) (string,error)
	GetAll() ([]string,error)
}

type MultiServerDiscovery struct {
	r *rand.Rand //随机数怎么用
	index int
	servers []string
	mu sync.RWMutex
}

func NewMultiServerDiscovery(servers []string) *MultiServerDiscovery {
	d := &MultiServerDiscovery{
		r:       rand.New(rand.NewSource(time.Now().UnixNano())),
		servers: servers,
	}
	d.index = d.r.Intn(math.MaxInt32-1)
	return d
}

var _ Discovery = (*MultiServerDiscovery)(nil)

func (m MultiServerDiscovery) Refresh() error {
	return nil
}

func (m MultiServerDiscovery) Update(servers []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.servers = servers
	return nil
}

func (m MultiServerDiscovery) Get(mode SelectMode) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	n := len(m.servers)
	if n == 0 {
		return "",errors.New("rpc discovery:no available servers")
	}
	var s string
	switch mode {
	case RoundRobinSelect:
		s = m.servers[m.index%n]
		m.index = (m.index+1)%n
	case RandomSelect:
		s = m.servers[m.r.Intn(n)]
	default:
		return "",errors.New("rpc discovery:invalid select mode")
	}
	return s,nil
}

func (m MultiServerDiscovery) GetAll() ([]string, error) {
	m.mu.RLock()
	servers := make([]string,len(m.servers))
	copy(servers,m.servers)
	defer m.mu.RUnlock()
	return servers,nil
}

