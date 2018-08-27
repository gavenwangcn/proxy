package lb

import (
	"github.com/hotwheels/gateway/pkg/util"

	"github.com/valyala/fasthttp"
)

// HashRobin round robin loadBalance impl
type HashRobin struct {
}

// NewHashRobin create a RoundRobin
func NewHashRobin() LoadBalance {

	return HashRobin{}
}

// Select select a server from servers using RoundRobin
func (rr HashRobin) Select(req *fasthttp.Request, servers *util.ConsistentList) string {
	l := uint64(servers.Len())

	if 0 >= l {
		return ""
	}
	//获取 urlhash 的server
	s := servers.GetHashServer(string(req.URI().RequestURI()))

	return s
}
