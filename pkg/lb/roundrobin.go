package lb

import (
	"sync/atomic"

	"github.com/hotwheels/gateway/pkg/util"

	"github.com/valyala/fasthttp"
)

// RoundRobin round robin loadBalance impl
type RoundRobin struct {
	ops *uint64
}

// NewRoundRobin create a RoundRobin
func NewRoundRobin() LoadBalance {
	var ops uint64
	ops = 0

	return RoundRobin{
		ops: &ops,
	}
}

// Select select a server from servers using RoundRobin
func (rr RoundRobin) Select(req *fasthttp.Request, servers *util.ConsistentList) string {
	l := uint64(servers.Len())

	if 0 >= l {
		return ""
	}

	index := int(atomic.AddUint64(rr.ops, 1) % l)
	i := 0
	for iter := servers.Front(); iter != nil; iter = iter.Next() {
		if i == index {
			return iter.Value.(string)
		}

		i++
	}
	return ""
}
