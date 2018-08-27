package model

import (
	"encoding/json"
	"io"
	"sync"

	"github.com/CodisLabs/codis/pkg/utils/log"
	"github.com/hotwheels/gateway/pkg/lb"
	"github.com/hotwheels/gateway/pkg/util"
	"github.com/valyala/fasthttp"
)

// Cluster cluster
type Cluster struct {
	Key         string   `json:"key, omitempty"`
	Name        string   `json:"name,omitempty"`
	LbName      string   `json:"lbName,omitempty"`
	BindServers []string `json:"bindServers,omitempty"`

	svrs   *util.ConsistentList
	rwLock *sync.RWMutex
	lb     lb.LoadBalance
}

// UnMarshalCluster unmarshal
func UnMarshalCluster(data []byte) (*Cluster, error) {
	v := &Cluster{}
	err := json.Unmarshal(data, v)

	if err != nil {
		return nil, err
	}
	v.init()

	return v, nil
}

// UnMarshalDelCluster unmarshal
func UnMarshalDelCluster(path, key string) *Cluster {
	v := &Cluster{Key: path, Name: key}
	return v
}

// UnMarshalClusterFromReader unmarshal from reader
func UnMarshalClusterFromReader(r io.Reader) (*Cluster, error) {
	v := &Cluster{}

	decoder := json.NewDecoder(r)
	err := decoder.Decode(v)

	if nil != err {
		return nil, err
	}

	return v, v.init()
}

// NewCluster create a cluster
func NewCluster(name string, lbName string) (*Cluster, error) {
	c := &Cluster{
		Name:   name,
		LbName: lbName,
	}

	return c, c.init()
}

// AddBind add bind
func (c *Cluster) AddBind(bind *Bind) {
	index := c.indexOf(bind.ServerAddr)
	if index == -1 {
		c.BindServers = append(c.BindServers, bind.ServerAddr)
	}
}

// HasBind add bind
func (c *Cluster) HasBind() bool {
	return len(c.BindServers) > 0
}

// RemoveBind remove bind
func (c *Cluster) RemoveBind(serverAddr string) {
	index := c.indexOf(serverAddr)
	if index >= 0 {
		c.BindServers = append(c.BindServers[:index], c.BindServers[index+1:]...)
	}
}

func (c *Cluster) indexOf(serverAddr string) int {
	for index, s := range c.BindServers {
		if s == serverAddr {
			return index
		}
	}

	return -1
}

func (c *Cluster) init() error {
	c.svrs = util.NewConsistentList()
	c.lb = lb.NewLoadBalance(c.LbName)
	c.rwLock = &sync.RWMutex{}

	return nil
}

func (c *Cluster) updateFrom(cluster *Cluster) {
	if c.rwLock != nil {
		c.rwLock.Lock()
		defer c.rwLock.Unlock()
	}

	c.LbName = cluster.LbName
	c.lb = lb.NewLoadBalance(c.LbName)

	log.Infof("Cluster <%s> updated, %+v", c.Name, c)
}

func (c *Cluster) doInEveryBindServers(callback func(string)) {
	c.rwLock.Lock()
	defer c.rwLock.Unlock()

	for iter := c.svrs.Back(); iter != nil; iter = iter.Prev() {
		addr, _ := iter.Value.(string)
		callback(addr)
	}
}

func (c *Cluster) unbind(svr *Server) {
	c.rwLock.Lock()
	defer c.rwLock.Unlock()

	c.doUnBind(svr)
}

func (c *Cluster) doUnBind(svr *Server) {
	util.RemoveOfConList(c.svrs, svr.Addr)
	log.Infof("UnBind <%s,%s> succ.", svr.Addr, c.Name)
}

func (c *Cluster) bind(svr *Server) {
	c.rwLock.Lock()
	defer c.rwLock.Unlock()

	if util.IndexOfConList(c.svrs, svr.Addr) >= 0 {
		log.Infof("Bind <%s,%s> already created.", svr.Addr, c.Name)
		return
	}

	c.svrs.Put(svr.Addr)

	log.Infof("Bind <%s,%s> created.", svr.Addr, c.Name)
}

// Select return a server using spec loadbalance
func (c *Cluster) Select(req *fasthttp.Request) string {
	c.rwLock.RLock()
	defer c.rwLock.RUnlock()

	s := c.lb.Select(req, c.svrs)

	return s
}

// Marshal marshal
func (c *Cluster) Marshal() []byte {
	v, _ := json.Marshal(c)
	return v
}
