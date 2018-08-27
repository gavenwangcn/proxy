package lb

import (
	"github.com/hotwheels/gateway/pkg/util"
	"github.com/valyala/fasthttp"
)

const (
	// ROUNDROBIN round robin
	ROUNDROBIN = "ROUNDROBIN"
	HASHROBIN  = "HASHROBIN"
)

var (
	supportLbs = []string{ROUNDROBIN, HASHROBIN}
)

var (
	// LBS map loadBalance name and process function
	LBS = map[string]func() LoadBalance{
		ROUNDROBIN: NewRoundRobin,
		HASHROBIN:  NewHashRobin,
	}
)

// LoadBalance loadBalance interface
type LoadBalance interface {
	Select(req *fasthttp.Request, servers *util.ConsistentList) string
}

// GetSupportLBS return supported loadBalances
func GetSupportLBS() []string {
	return supportLbs
}

// NewLoadBalance create a LoadBalance
func NewLoadBalance(name string) LoadBalance {
	return LBS[name]()
}
