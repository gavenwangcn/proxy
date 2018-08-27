// local @15070229 本地启动模式
package model

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/CodisLabs/codis/pkg/utils/log"
)

type LocalStore struct {
	prefix            string
	clustersDir       string
	serversDir        string
	bindsDir          string
	apisDir           string
	proxiesDir        string
	routingsDir       string
	clustersDirCNum   int32
	serversDirCNum    int32
	bindsDirCNum      int32
	apisDirCNum       int32
	routingsDirCNum   int32
	deleteServersDir  string
	deleteClustersDir string
	deleteAPIsDir     string
}

// NewLocStore create a local store
func NewLocalStore(prefix string) (Store, error) {
	store := new(LocalStore)
	prefix = strings.TrimPrefix(prefix, "/")
	store.prefix = prefix
	store.clustersDir = fmt.Sprintf("%s/clusters", prefix)
	store.serversDir = fmt.Sprintf("%s/servers", prefix)
	store.bindsDir = fmt.Sprintf("%s/binds", prefix)
	store.apisDir = fmt.Sprintf("%s/apis", prefix)
	store.proxiesDir = fmt.Sprintf("%s/proxy", prefix)
	store.routingsDir = fmt.Sprintf("%s/routings", prefix)
	store.deleteServersDir = fmt.Sprintf("%s/delete/servers", prefix)
	store.deleteClustersDir = fmt.Sprintf("%s/delete/clusters", prefix)
	store.deleteAPIsDir = fmt.Sprintf("%s/delete/apis", prefix)
	store.clustersDirCNum = 0
	store.serversDirCNum = 0
	store.bindsDirCNum = 0
	store.apisDirCNum = 0
	store.routingsDirCNum = 0

	return store, nil
}

// @15070229 本地启动忽略增删改，只做读
// SaveAPI save a api in store
func (l *LocalStore) SaveAPI(api *API) error {
	return nil
}

// UpdateAPI update a api in store
func (l *LocalStore) UpdateAPI(api *API) error {
	return nil
}

// DeleteAPI delete a api from store
func (l *LocalStore) DeleteAPI(apiURL string) error {
	return nil
}

// GetAPIs return api list from store
func (l *LocalStore) GetAPIs() ([]*API, error) {
	rd, err := ioutil.ReadDir(l.apisDir)

	if nil != err {
		return nil, err
	}

	apis := []*API{}
	i := 0
	for _, fi := range rd {
		if !fi.IsDir() &&
			strings.HasSuffix(fi.Name(), ".json") {
			apiDir := fmt.Sprintf("%s/%s", l.apisDir, fi.Name())
			rs, rerr := ioutil.ReadFile(apiDir)
			if nil != rerr {
				return nil, rerr
			}
			if len(rs) == 0 || rs == nil {
				continue
			}
			v, marshalerr := UnMarshalAPI(rs)
			if marshalerr != nil {
				log.Errorf("GetAPIs marshal api value is error %s", string(rs))
				continue
			}
			apis = append(apis, v)
			i = i + 1
		}
	}
	l.apisDirCNum = int32(i + 1)

	return apis, nil
}

// GetAPI return api by url from store
func (l *LocalStore) GetAPI(apiURL string) (*API, error) {
	return nil, nil
}

// SaveServer save a server to store
func (l *LocalStore) SaveServer(svr *Server) error {
	return nil
}

// UpdateServer update a server to store
func (l *LocalStore) UpdateServer(svr *Server) error {
	return nil
}

// DownServer update a server to store
func (l *LocalStore) DownServer(addr string) error {
	return nil
}

// DeleteServer delete a server from store
func (l *LocalStore) DeleteServer(addr string) error {
	return nil
}

// GetServers return server from store
func (l *LocalStore) GetServers() ([]*Server, error) {
	rd, err := ioutil.ReadDir(l.serversDir)
	if nil != err {
		return nil, err
	}

	servers := []*Server{}
	i := 0
	for _, fi := range rd {
		if !fi.IsDir() &&
			strings.HasSuffix(fi.Name(), ".json") {
			serverDir := fmt.Sprintf("%s/%s", l.serversDir, fi.Name())
			rs, rerr := ioutil.ReadFile(serverDir)
			if nil != rerr {
				return nil, rerr
			}
			if len(rs) == 0 || rs == nil {
				continue
			}
			s, marshalerr := UnMarshalServer(rs)
			if marshalerr != nil {
				log.Errorf("GetServer marshal server value is error %s", string(rs))
				continue
			}
			servers = append(servers, s)
			i = i + 1
		}
	}
	l.serversDirCNum = int32(i + 1)

	log.Infof("GetServers From Local OK")

	return servers, nil

}

// GetServer return spec server
func (l *LocalStore) GetServer(serverAddr string) (*Server, error) {
	return nil, nil
}

// SaveCluster save a cluster to store
func (l *LocalStore) SaveCluster(cluster *Cluster) error {
	return nil
}

// UpdateCluster update a cluster to store
func (l *LocalStore) UpdateCluster(cluster *Cluster) error {
	return nil
}

// DeleteCluster delete a cluster from store
func (l *LocalStore) DeleteCluster(name string) error {
	return nil
}

// GetClusters return clusters in store
func (l *LocalStore) GetClusters() ([]*Cluster, error) {
	rd, err := ioutil.ReadDir(l.clustersDir)
	if nil != err {
		return nil, err
	}

	clusters := []*Cluster{}
	i := 0
	for _, fi := range rd {
		if !fi.IsDir() &&
			strings.HasSuffix(fi.Name(), ".json") {
			clusterDir := fmt.Sprintf("%s/%s", l.clustersDir, fi.Name())
			rs, rerr := ioutil.ReadFile(clusterDir)
			if nil != rerr {
				return nil, rerr
			}
			if len(rs) == 0 || rs == nil {
				continue
			}
			s, marshalerr := UnMarshalCluster(rs)
			if marshalerr != nil {
				log.Errorf("GetClusters marshal cluster value is error %s", string(rs))
				continue
			}
			clusters = append(clusters, s)
			i = i + 1
		}
	}
	l.clustersDirCNum = int32(i + 1)

	log.Infof("GetClusters From Local OK")

	return clusters, nil
}

// GetCluster return cluster info
func (l *LocalStore) GetCluster(clusterName string) (*Cluster, error) {
	return nil, nil
}

// SaveBind save bind to store
func (l *LocalStore) SaveBind(bind *Bind) error {
	return nil
}

// UnBind delete bind from store
func (l *LocalStore) UnBind(bind *Bind) error {
	return nil
}

// GetBinds return binds info
func (l *LocalStore) GetBinds() ([]*Bind, error) {
	rd, err := ioutil.ReadDir(l.bindsDir)
	if nil != err {
		return nil, err
	}

	binds := []*Bind{}
	i := 0
	for _, fi := range rd {
		if !fi.IsDir() &&
			strings.HasSuffix(fi.Name(), ".json") {
			infos := strings.SplitN(fi.Name(), "-", 2)
			serverAddr := strings.Replace(infos[0], "_", ":", -1)
			clusterName := strings.TrimRight(infos[1], ".json")
			b := &Bind{
				Key:         fmt.Sprintf("%s/%s-%s", l.bindsDir, serverAddr, clusterName),
				ServerAddr:  serverAddr,
				ClusterName: clusterName,
			}
			binds = append(binds, b)
			i = i + 1
		}
	}
	l.bindsDirCNum = int32(i + 1)

	log.Infof("GetBinds From Local OK")

	return binds, nil
}

// SaveRouting save route to store
func (l *LocalStore) SaveRouting(routing *Routing) error {
	return nil
}

// GetRoutings return routes in store
func (l *LocalStore) GetRoutings() ([]*Routing, error) {
	rd, err := ioutil.ReadDir(l.routingsDir)
	if nil != err {
		return nil, err
	}

	routings := []*Routing{}
	i := 0
	for _, fi := range rd {
		if !fi.IsDir() &&
			strings.HasSuffix(fi.Name(), ".json") {
			routingDir := fmt.Sprintf("%s/%s", l.routingsDir, fi.Name())
			rs, rerr := ioutil.ReadFile(routingDir)
			if nil != rerr {
				return nil, rerr
			}
			if len(rs) == 0 || rs == nil {
				continue
			}
			r, marshalerr := UnMarshalRouting(rs)
			if marshalerr != nil {
				log.Errorf("GetRoutings marshal routing value is error %s", string(rs))
				continue
			}
			routings = append(routings, r)
			i = i + 1
		}
	}
	l.routingsDirCNum = int32(i + 1)

	log.Infof("GetClusters From Local OK")

	return routings, nil
}

// Clean clean data in store
func (l *LocalStore) Clean() error {

	return nil
}

// GC exec gc, delete some data
func (l *LocalStore) GC() error {
	return nil
}

// Watch watch event from zk
func (l *LocalStore) Watch(evtCh chan *Evt, stopCh chan bool) error {
	return nil
}

func (l *LocalStore) Close() {
}
