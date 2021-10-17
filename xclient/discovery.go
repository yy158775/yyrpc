package xclient

import (
	"errors"
	"math"
	"math/rand"
	"sync"
	"time"
)

type SelectMode int

//注意用法都会不写了吗？
const (
	RoundRobinSelect SelectMode = iota
	RandomSelect
)

type Discovery interface {
	Refresh() error
	Update(servers []string) error
	Get(mode SelectMode) (string,error)  //这一点可以思考清楚，到底吧selectmode以什么样的方式传入
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
	//r 是一个产生随机数的实例，初始化时使用时间戳设定随机数种子，避免每次产生相同的随机数序列.
	//index 记录 Round Robin 算法已经轮询到的位置，为了避免每次从 0 开始，初始化时随机设定一个值。

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

//像获得全部，怎么给他，能不能给
//用户如果想获得，这里不能全给他
func (m MultiServerDiscovery) GetAll() ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	servers := make([]string,len(m.servers))
	copy(servers,m.servers)
	return servers,nil
}

