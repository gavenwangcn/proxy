// local_test.go
package model

import (
	"testing"
)

func TestSaveApi(t *testing.T) {
	store, err := NewLocalStore("/gateway")
	if nil != err {
		t.Error("local store new local store err.")
		return
	}
	api := new(API)
	err1 := store.SaveAPI(api)
	if nil != err1 {
		t.Error("local store save api err.")
		return
	}
}

func TestGetApis(t *testing.T) {
	store, err := NewLocalStore("D:/go/src/github.com/hotwheels/gateway/cmd/proxy/gateway")
	if nil != err {
		t.Error("local store new local store err.")
		return
	}
	apis, err1 := store.GetAPIs()
	if nil != err1 {
		t.Error("local store get apis err.")
		return
	}
	for _, n := range apis {
		t.Logf("api:[%s,%s,%s,%s,%s,%s]",
			n.Key,
			n.URL,
			n.Method,
			n.Status,
			n.CustomHeaders,
			n.Desc)
	}

}

func TestGetServers(t *testing.T) {
	store, err := NewLocalStore("D:/go/src/github.com/hotwheels/gateway/cmd/proxy/gateway")
	if nil != err {
		t.Error("local store new local store err.")
		return
	}
	servers, err1 := store.GetServers()
	if nil != err1 {
		t.Error("local store get servers err.")
		return
	}
	for _, n := range servers {
		t.Logf("server:[%s,%s,%s,%s,%s,%s]",
			n.Key,
			n.Schema,
			n.Addr,
			n.BindClusters,
			n.MaxQPS,
			n.CheckPath)
	}
}

func TestGetClusters(t *testing.T) {
	store, err := NewLocalStore("D:/go/src/github.com/hotwheels/gateway/cmd/proxy/gateway")
	if nil != err {
		t.Error("local store new local store err.")
		return
	}
	clusters, err1 := store.GetClusters()
	if nil != err1 {
		t.Error("local store get clusters err.")
		return
	}
	for _, n := range clusters {
		t.Logf("cluster:[%s,%s,%s,%s,%s,%s]",
			n.Key,
			n.Name,
			n.LbName,
			n.BindServers)
	}
}

func TestGetBinds(t *testing.T) {
	store, err := NewLocalStore("D:/go/src/github.com/hotwheels/gateway/cmd/proxy/gateway")
	if nil != err {
		t.Error("local store new local store err.")
		return
	}
	binds, err1 := store.GetBinds()
	if nil != err1 {
		t.Error("local store get binds err.")
		return
	}
	for _, n := range binds {
		t.Logf("binds:[%s,%s,%s]",
			n.Key,
			n.ServerAddr,
			n.ClusterName)
	}
}

func TestGetRoutings(t *testing.T) {
	store, err := NewLocalStore("D:/go/src/github.com/hotwheels/gateway/cmd/proxy/gateway")
	if nil != err {
		t.Error("local store new local store err.")
		return
	}
	routings, err1 := store.GetRoutings()
	if nil != err1 {
		t.Error("local store get binds err.")
		return
	}
	for _, n := range routings {
		t.Logf("binds:[%s,%s,%s,%s]",
			n.Key,
			n.ClusterName,
			n.URL,
			n.Cfg)
	}
}
