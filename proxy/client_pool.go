package proxy

import (
	"sync"
	"time"

	"github.com/hotwheels/gateway/conf"
)

const DefaultFlushTime = time.Duration(5) * time.Second

// NewClientPool fast http client
type ClientPool struct {
	conf *conf.Conf

	poolLock sync.Mutex
	pools    map[string]*DefaultClient
}

// NewClientPool create NewClientPool instance
func NewClientPool(conf *conf.Conf) *ClientPool {
	clientPool := &ClientPool{
		conf:  conf,
		pools: make(map[string]*DefaultClient),
	}

	return clientPool
}

func (cp *ClientPool) GetDefaultClient(addr string) *DefaultClient {
	var fc *DefaultClient

	cp.poolLock.Lock()
	fc = cp.pools[addr]
	if fc == nil {
		newfc := NewDefaultClient(cp.conf, addr)
		cp.pools[addr] = newfc
		cp.poolLock.Unlock()
		return newfc
	}
	cp.poolLock.Unlock()
	return fc
}
