package util

import (
	"hash/crc32"
	"sort"
	"strconv"
	"sync"
)

const DEFAULT_REPLICAS = 10

type HashRing []uint32

func (c HashRing) Len() int {
	return len(c)
}

func (c HashRing) Less(i, j int) bool {
	return c[i] < c[j]
}

func (c HashRing) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

type Node struct {
	Ip     string
	Weight int
}

func NewNode(ip string, weight int) *Node {
	return &Node{
		Ip:     ip,
		Weight: weight,
	}
}

type Consistent struct {
	Nodes     map[uint32]Node
	numReps   int
	Resources map[string]bool
	ring      HashRing
	sync.RWMutex
}

func NewConsistent() *Consistent {
	return new(Consistent).Init()
}

func (c *Consistent) Init() *Consistent {
	c.Nodes = make(map[uint32]Node)
	c.numReps = DEFAULT_REPLICAS
	c.Resources = make(map[string]bool)
	c.ring = HashRing{}
	return c
}

func (c *Consistent) AddCon(node *Node) bool {
	c.Lock()
	defer c.Unlock()

	if _, ok := c.Resources[node.Ip]; ok {
		return false
	}

	count := c.numReps * node.Weight
	for i := 0; i < count; i++ {
		str := c.joinStr(i, node)
		c.Nodes[c.hashStr(str)] = *(node)
	}
	c.Resources[node.Ip] = true
	c.sortHashRing()
	return true
}

func (c *Consistent) sortHashRing() {
	c.ring = HashRing{}
	for k := range c.Nodes {
		c.ring = append(c.ring, k)
	}
	sort.Sort(c.ring)
}

func (c *Consistent) joinStr(i int, node *Node) string {
	return node.Ip + "*" + strconv.Itoa(node.Weight) +
		"-" + strconv.Itoa(i)
}

// MurMurHash算法 :https://github.com/spaolacci/murmur3
func (c *Consistent) hashStr(key string) uint32 {
	return crc32.ChecksumIEEE([]byte(key))
}

func (c *Consistent) GetCon(key string) Node {
	c.RLock()
	defer c.RUnlock()

	hash := c.hashStr(key)
	i := c.search(hash)

	return c.Nodes[c.ring[i]]
}

func (c *Consistent) search(hash uint32) int {

	i := sort.Search(len(c.ring), func(i int) bool { return c.ring[i] >= hash })
	if i < len(c.ring) {
		if i == len(c.ring)-1 {
			return 0
		} else {
			return i
		}
	} else {
		return len(c.ring) - 1
	}
}

func (c *Consistent) RemoveCon(node *Node) {
	c.Lock()
	defer c.Unlock()

	if _, ok := c.Resources[node.Ip]; !ok {
		return
	}

	delete(c.Resources, node.Ip)

	count := c.numReps * node.Weight
	for i := 0; i < count; i++ {
		str := c.joinStr(i, node)
		delete(c.Nodes, c.hashStr(str))
	}
	c.sortHashRing()
}
