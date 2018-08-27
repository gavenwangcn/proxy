package util

import (
	"container/list"

	"github.com/CodisLabs/codis/pkg/utils/errors"
)

type ConsistentList struct {
	*list.List
	*Consistent
}

func NewConsistentList() *ConsistentList {
	return &ConsistentList{list.New(), NewConsistent()}
}

func (this *ConsistentList) Put(value string) {
	this.PushBack(value)
	ip := value
	node := NewNode(ip, 4)
	this.AddCon(node)
}

func (this *ConsistentList) GetHashServer(key string) string {
	n := this.GetCon(key)
	return n.Ip

}

func (this *ConsistentList) RemoveServer(value interface{}) error {
	if s, ok := value.(string); ok {
		node := NewNode(s, 4)
		this.RemoveCon(node)
		return nil
	} else {
		return errors.New("error RemoveServer the value is not a server")
	}
}
