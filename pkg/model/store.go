package model

import (
	"encoding/base64"
	"strings"

	"github.com/CodisLabs/codis/pkg/utils/errors"
)

// EvtType event type
type EvtType int

// EvtSrc event src
type EvtSrc int

const (
	// EventTypeNew event type new
	EventTypeNew = EvtType(0)
	// EventTypeUpdate event type update
	EventTypeUpdate = EvtType(1)
	// EventTypeDelete event type delete
	EventTypeDelete = EvtType(2)
	// EventTypeReload event type Reload
	EventTypeReload = EvtType(3)
)

const (
	// EventSrcCluster cluster event
	EventSrcCluster = EvtSrc(0)
	// EventSrcServer server event
	EventSrcServer = EvtSrc(1)
	// EventSrcBind bind event
	EventSrcBind = EvtSrc(2)
	// EventSrcAPI api event
	EventSrcAPI = EvtSrc(3)
	// EventSrcRouting routing event
	EventSrcRouting = EvtSrc(4)

	// Error event
	EventError = EvtSrc(5)
)

// Evt event
type Evt struct {
	Src   EvtSrc
	Type  EvtType
	Key   string
	Value interface{}
}

// Store store interface
type Store interface {
	SaveBind(bind *Bind) error
	UnBind(bind *Bind) error
	GetBinds() ([]*Bind, error)

	SaveCluster(cluster *Cluster) error
	UpdateCluster(cluster *Cluster) error
	DeleteCluster(name string) error
	GetClusters() ([]*Cluster, error)
	GetCluster(clusterName string) (*Cluster, error)

	SaveServer(svr *Server) error
	UpdateServer(svr *Server) error
	DeleteServer(addr string) error
	GetServers() ([]*Server, error)
	GetServer(serverAddr string) (*Server, error)
	DownServer(addr string) error

	SaveAPI(api *API) error
	UpdateAPI(api *API) error
	DeleteAPI(url string) error
	GetAPIs() ([]*API, error)
	GetAPI(url string) (*API, error)

	SaveRouting(routing *Routing) error
	GetRoutings() ([]*Routing, error)

	Watch(evtCh chan *Evt, stopCh chan bool) error

	Clean() error
	GC() error
	Close()
}

var ErrUnknownCoordinator = errors.New("unknown coordinator")

func NewStore(coordinator string, etcdAddrs []string,
	zkAddrs string, prefix string) (Store, error) {
	switch coordinator {
	case "zk", "zookeeper":
		return NewZkStore(zkAddrs, prefix)
	case "etcd":
		return NewEtcdStore(etcdAddrs, prefix)
	//@15070229 Start from local
	case "local":
		return NewLocalStore(prefix)
	default:
		return NewLocalStore("gateway")
	}
	return nil, errors.Trace(ErrUnknownCoordinator)
}

func getAPIKey(apiURL string) string {
	url := strings.Replace(apiURL, "/", "-", -1)
	//修改成urlencoding
	return base64.URLEncoding.EncodeToString([]byte(url))
}

func parseAPIKey(key string) string {
	raw := decodeAPIKey(key)
	return raw
}

func decodeAPIKey(key string) string {
	//修改成urlencoding
	raw, _ := base64.URLEncoding.DecodeString(key)
	return string(raw)
}
