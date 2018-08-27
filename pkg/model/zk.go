package model

import (
	"fmt"
	"strings"

	"github.com/CodisLabs/codis/pkg/utils/errors"
	"github.com/CodisLabs/codis/pkg/utils/log"
	"github.com/samuel/go-zookeeper/zk"
)

// zkStore zk store impl
type ZkStore struct {
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

	cli    *ZkClient
	closed bool

	watchCh    chan zk.Event
	watchProxy chan zk.Event
	evtCh      chan *Evt
	proxyInfo  *ProxyInfo

	watchMethodMapping map[EvtSrc]func(EvtType, string) *Evt
}

var ErrNodeNullValue = errors.New("zk node value is null")

// NewZkStore create a etcd store
func NewZkStore(zkAddrs string, prefix string) (Store, error) {
	client, err := New(zkAddrs, -1)
	if err != nil {
		return nil, err
	}
	store := new(ZkStore)
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
	store.closed = false
	store.cli = client
	store.watchMethodMapping = make(map[EvtSrc]func(EvtType, string) *Evt)
	store.init()
	return store, nil
}

// SaveAPI save a api in store
func (z *ZkStore) SaveAPI(api *API) error {
	key := fmt.Sprintf("%s/%s", z.apisDir, getAPIKey(api.URL))
	err := z.cli.Create(key, api.Marshal())

	return err
}

// UpdateAPI update a api in store
func (z *ZkStore) UpdateAPI(api *API) error {
	key := fmt.Sprintf("%s/%s", z.apisDir, getAPIKey(api.URL))
	err := z.cli.Update(key, api.Marshal())

	return err
}

// DeleteAPI delete a api from store
func (z *ZkStore) DeleteAPI(apiURL string) error {
	return z.deleteKey(getAPIKey(apiURL), z.apisDir, z.deleteAPIsDir)
}

func (z *ZkStore) deleteAPIGC(key string) error {
	return z.deleteKey(key, z.apisDir, z.deleteAPIsDir)
}

func (z *ZkStore) deleteKey(value, prefixKey, cacheKey string) error {
	deleteKey := fmt.Sprintf("%s/%s", cacheKey, value)
	err := z.cli.Create(deleteKey, []byte(value))

	if err != nil {
		return err
	}

	key := fmt.Sprintf("%s/%s", prefixKey, value)
	err = z.cli.Delete(key)

	if nil != err {
		return err
	}

	err = z.cli.Delete(deleteKey)

	return err
}

// GetAPIs return api list from store
func (z *ZkStore) GetAPIs() ([]*API, error) {
	numC, ls, err := z.cli.List(z.apisDir, false)
	if nil != err {
		return nil, err
	}
	z.apisDirCNum = numC
	l := len(ls)
	apis := make([]*API, l)

	for i := 0; i < l; i++ {
		rs, rerr := z.cli.Read(ls[i], false)
		if rerr != nil {
			return nil, rerr
		}
		if rs == nil {
			continue
		}
		v, marshalerr := UnMarshalAPI(rs)
		if marshalerr != nil {
			log.Errorf("GetAPIs marshal api value is error %s", string(rs))
			continue
		}
		apis[i] = v
		v.Key = ls[i]
	}

	return apis, nil
}

// GetAPI return api by url from store
func (z *ZkStore) GetAPI(apiURL string) (*API, error) {
	key := fmt.Sprintf("%s/%s", z.apisDir, getAPIKey(apiURL))
	rs, err := z.cli.Read(key, false)

	if nil != err {
		return nil, err
	}
	if rs == nil {
		return nil, errors.Trace(ErrNodeNullValue)
	}
	v, marshalerr := UnMarshalAPI(rs)
	if marshalerr != nil {
		log.Errorf("GetAPI marshal api value is error %s", string(rs))
		return nil, marshalerr
	}
	v.Key = key
	return v, nil
}

// SaveServer save a server to store
func (z *ZkStore) SaveServer(svr *Server) error {
	key := fmt.Sprintf("%s/%s", z.serversDir, svr.Addr)
	err := z.cli.Create(key, svr.Marshal())

	return err
}

// UpdateServer update a server to store
func (z *ZkStore) UpdateServer(svr *Server) error {
	old, err := z.GetServer(svr.Addr)

	if nil != err {
		return err
	}

	old.updateFrom(svr)

	return z.doUpdateServer(old)
}

// DownServer update a server to store
func (z *ZkStore) DownServer(addr string) error {
	old, err := z.GetServer(addr)

	if nil != err {
		return err
	}
	old.Status = Down
	return z.doUpdateServer(old)
}

func (z *ZkStore) doUpdateServer(svr *Server) error {
	key := fmt.Sprintf("%s/%s", z.serversDir, svr.Addr)
	err := z.cli.Update(key, svr.Marshal())

	return err
}

// DeleteServer delete a server from store
func (z *ZkStore) DeleteServer(addr string) error {
	svr, err := z.GetServer(addr)
	if err != nil {
		return err
	}

	if svr.HasBind() {
		return ErrHasBind
	}

	return z.deleteKey(addr, z.serversDir, z.deleteServersDir)
}

// GetServers return server from store
func (z *ZkStore) GetServers() ([]*Server, error) {
	numC, ls, err := z.cli.List(z.serversDir, false)

	if nil != err {
		return nil, err
	}
	z.serversDirCNum = numC
	l := len(ls)
	servers := make([]*Server, l)
	for i := 0; i < l; i++ {
		rs, rerr := z.cli.Read(ls[i], false)
		if rerr != nil {
			return nil, rerr
		}
		if rs == nil {
			continue
		}
		v, marshalerr := UnMarshalServer(rs)
		if marshalerr != nil {
			log.Errorf("GetServers marshal server value is error %s", string(rs))
			continue
		}
		servers[i] = v
		v.Key = ls[i]
	}

	return servers, nil
}

// GetServer return spec server
func (z *ZkStore) GetServer(serverAddr string) (*Server, error) {
	key := fmt.Sprintf("%s/%s", z.serversDir, serverAddr)
	rs, err := z.cli.Read(key, false)

	if nil != err {
		return nil, err
	}

	if rs == nil {
		return nil, errors.Trace(ErrNodeNullValue)
	}
	v, marshalerr := UnMarshalServer(rs)
	if marshalerr != nil {
		log.Errorf("GetServer marshal server value is error %s", string(rs))
		return nil, marshalerr
	}
	v.Key = key
	return v, nil
}

// SaveCluster save a cluster to store
func (z *ZkStore) SaveCluster(cluster *Cluster) error {
	key := fmt.Sprintf("%s/%s", z.clustersDir, cluster.Name)
	err := z.cli.Create(key, cluster.Marshal())

	return err
}

// UpdateCluster update a cluster to store
func (z *ZkStore) UpdateCluster(cluster *Cluster) error {
	old, err := z.GetCluster(cluster.Name)

	if nil != err {
		return err
	}

	old.updateFrom(cluster)

	return z.doUpdateCluster(old)
}

func (z *ZkStore) doUpdateCluster(cluster *Cluster) error {
	key := fmt.Sprintf("%s/%s", z.clustersDir, cluster.Name)
	err := z.cli.Update(key, cluster.Marshal())

	return err
}

// DeleteCluster delete a cluster from store
func (z *ZkStore) DeleteCluster(name string) error {
	c, err := z.GetCluster(name)
	if err != nil {
		return err
	}

	if c.HasBind() {
		return ErrHasBind
	}

	return z.deleteKey(name, z.clustersDir, z.deleteClustersDir)
}

// GetClusters return clusters in store
func (z *ZkStore) GetClusters() ([]*Cluster, error) {
	numC, ls, err := z.cli.List(z.clustersDir, false)
	if nil != err {
		return nil, err
	}
	z.clustersDirCNum = numC
	l := len(ls)
	clusters := make([]*Cluster, l)

	for i := 0; i < l; i++ {
		log.Infof("GetClusters path %s", ls[i])
		rs, rerr := z.cli.Read(ls[i], true)
		log.Infof("GetClusters rs %s", string(rs))
		if rerr != nil {
			return nil, rerr
		}
		if rs == nil {
			continue
		}
		v, marshalerr := UnMarshalCluster(rs)
		if marshalerr != nil {
			log.Errorf("GetClusters marshal cluster value is error %s", string(rs))
			continue
		}
		clusters[i] = v
		v.Key = ls[i]
	}

	return clusters, nil
}

// GetCluster return cluster info
func (z *ZkStore) GetCluster(clusterName string) (*Cluster, error) {
	key := fmt.Sprintf("%s/%s", z.clustersDir, clusterName)
	rs, err := z.cli.Read(key, false)

	if nil != err {
		return nil, err
	}
	if rs == nil {
		return nil, errors.Trace(ErrNodeNullValue)
	}
	v, marshalerr := UnMarshalCluster(rs)
	if marshalerr != nil {
		log.Errorf("GetCluster marshal cluster value is error %s", string(rs))
		return nil, marshalerr
	}
	v.Key = key
	return v, nil
}

// SaveBind save bind to store
func (z *ZkStore) SaveBind(bind *Bind) error {
	key := fmt.Sprintf("%s/%s", z.bindsDir, bind.ToString())
	err := z.cli.Create(key, []byte(""))

	if err != nil {
		return err
	}

	// update server bind info
	svr, err := z.GetServer(bind.ServerAddr)
	if err != nil {
		return err
	}

	svr.AddBind(bind)

	err = z.doUpdateServer(svr)
	if err != nil {
		return err
	}

	// update cluster bind info
	c, err := z.GetCluster(bind.ClusterName)
	if err != nil {
		return err
	}

	c.AddBind(bind)

	return z.doUpdateCluster(c)
}

// UnBind delete bind from store
func (z *ZkStore) UnBind(bind *Bind) error {
	key := fmt.Sprintf("%s/%s", z.bindsDir, bind.ToString())

	err := z.cli.Delete(key)
	if err != nil {
		return err
	}

	svr, err := z.GetServer(bind.ServerAddr)
	if err != nil {
		return err
	}

	svr.RemoveBind(bind.ClusterName)

	err = z.doUpdateServer(svr)
	if err != nil {
		return err
	}

	c, err := z.GetCluster(bind.ClusterName)
	if err != nil {
		return err
	}

	c.RemoveBind(bind.ServerAddr)
	return z.doUpdateCluster(c)
}

// GetBinds return binds info
func (z *ZkStore) GetBinds() ([]*Bind, error) {
	numC, ls, err := z.cli.List(z.bindsDir, false)

	if nil != err {
		return nil, err
	}
	z.bindsDirCNum = numC
	l := len(ls)
	values := make([]*Bind, l)

	for i := 0; i < l; i++ {
		key := ls[i]
		infos := strings.SplitN(key, "-", 2)

		values[i] = &Bind{
			Key:         ls[i],
			ServerAddr:  strings.TrimLeft(infos[0], z.bindsDir+"/"),
			ClusterName: infos[1],
		}
	}

	return values, nil
}

// SaveRouting save route to store
func (z *ZkStore) SaveRouting(routing *Routing) error {
	key := fmt.Sprintf("%s/%s", z.routingsDir, routing.ID)
	err := z.cli.Create(key, routing.Marshal())

	return err
}

// GetRoutings return routes in store
func (z *ZkStore) GetRoutings() ([]*Routing, error) {
	numC, ls, err := z.cli.List(z.routingsDir, false)

	if nil != err {
		return nil, err
	}
	z.routingsDirCNum = numC
	l := len(ls)
	routings := make([]*Routing, l)

	for i := 0; i < l; i++ {
		rs, rerr := z.cli.Read(ls[i], false)
		if rerr != nil {
			return nil, rerr
		}
		if rs == nil {
			continue
		}
		v, marshalerr := UnMarshalRouting(rs)
		if marshalerr != nil {
			log.Errorf("GetRoutings marshal routing value is error %s", string(rs))
			continue
		}
		routings[i] = v
		v.Key = ls[i]
	}

	return routings, nil
}

// Clean clean data in store
func (z ZkStore) Clean() error {
	err := z.cli.Delete(z.prefix)

	return err
}

// GC exec gc, delete some data
func (z *ZkStore) GC() error {
	// process not complete delete opts
	err := z.gcDir(z.deleteServersDir, z.DeleteServer)

	if nil != err {
		return err
	}

	err = z.gcDir(z.deleteClustersDir, z.DeleteCluster)

	if nil != err {
		return err
	}

	err = z.gcDir(z.deleteAPIsDir, z.deleteAPIGC)

	if nil != err {
		return err
	}

	return nil
}

func (z *ZkStore) gcDir(dir string, fn func(value string) error) error {
	_, ls, err := z.cli.List(dir, false)
	if err != nil {
		return err
	}

	for _, l := range ls {
		err = fn(l)
		if err != nil {
			return err
		}
	}

	return nil
}

// Watch watch event from zk
func (z *ZkStore) Watch(evtCh chan *Evt, stopCh chan bool) error {

	z.watchCh = z.cli.events

	z.evtCh = evtCh

	go z.doWatch()

	return z.watchRegister()
}

func (z *ZkStore) watchRegister() error {

	cNumC, _, errc := z.cli.WatchCInData(z.clustersDir)
	if errc != nil {
		log.ErrorErrorf(errc, "zk watch error: <%s>", z.clustersDir)
		return errc
	}
	log.Infof("zk watch at: <%s>", z.clustersDir)
	z.clustersDirCNum = cNumC
	sNumC, _, errs := z.cli.WatchCInData(z.serversDir)
	if errs != nil {
		log.ErrorErrorf(errs, "zk watch error: <%s>", z.serversDir)
		return errs
	}
	log.Infof("zk watch at: <%s>", z.serversDir)
	z.serversDirCNum = sNumC
	bNumC, _, errb := z.cli.WatchCInData(z.bindsDir)
	log.Infof("zk watch at: <%s>", z.bindsDir)
	if errb != nil {
		return errb
	}
	z.bindsDirCNum = bNumC
	aNumC, _, erra := z.cli.WatchCInData(z.apisDir)
	if erra != nil {
		log.ErrorErrorf(erra, "zk watch error: <%s>", z.apisDir)
		return erra
	}
	log.Infof("zk watch at: <%s>", z.apisDir)
	z.apisDirCNum = aNumC
	rNumC, _, errr := z.cli.WatchCInData(z.routingsDir)
	if errr != nil {
		log.ErrorErrorf(errr, "zk watch error: <%s>", z.routingsDir)
		return errr
	}
	log.Infof("zk watch at: <%s>", z.routingsDir)
	z.routingsDirCNum = rNumC
	return nil
}

func (z *ZkStore) dispathEvent(change zk.Event) error {

	var evtSrc EvtSrc
	var evtType EvtType
	path := change.Path
	log.Debugf("dispathEvent path %s", path)
	var err error
	if change.Type == zk.EventSession {

		if change.State == zk.StateExpired {
			log.Infof("zk reconnect ok for re watchRegister")
			z.watchRegister()
		}
		return nil
	} else if change.Type == zk.EventNodeDeleted {
		evtType = EventTypeDelete
		evtSrc, _, err = z.getEvtSrc(path)
		if err != nil {
			return err
		}
	} else if change.Type == zk.EventNodeChildrenChanged {
		evtType = EventTypeReload
		evtSrc, _, err = z.getEvtSrc(path)
		if err != nil {
			return err
		}
		numC, errw := z.cli.WatchC(path)
		if errw != nil {
			log.ErrorErrorf(errw, "zk watchC error: <%s>", path)
			return errw
		}
		addChildFlag, _ := z.setNumCByPath(path, numC)
		if !addChildFlag {
			log.Debugf("addChildFlag false")
			return nil
		}
	} else if change.Type == zk.EventNodeDataChanged {
		evtType = EventTypeUpdate
		evtSrc, _, err = z.getEvtSrc(path)
		if err != nil {
			return err
		}
		errw := z.cli.WatchData(path)
		if errw != nil {
			log.ErrorErrorf(errw, "zk watchData error: <%s>", path)
		}
	} else {
		log.Debugf("zkclient dispathEvent not Used, Type <d%>", change.Type)
		return nil
	}

	log.Infof("zk changed: <%s>", path)

	z.evtCh <- z.watchMethodMapping[evtSrc](evtType, path)

	return nil
}

func (z *ZkStore) doWatch() {
	for {
		select {
		case change := <-z.watchCh:
			log.Debugf("dispathEvent: <%s>", change.Type)
			err := z.dispathEvent(change)
			if err != nil {
				log.Errorf("dispathEvent error <%s", err.Error())
			}
		}
	}
}

func (z *ZkStore) getEvtSrc(path string) (EvtSrc, string, error) {
	if strings.HasPrefix(path, z.clustersDir) {
		return EventSrcCluster, z.clustersDir, nil
	} else if strings.HasPrefix(path, z.serversDir) {
		return EventSrcServer, z.serversDir, nil
	} else if strings.HasPrefix(path, z.bindsDir) {
		return EventSrcBind, z.bindsDir, nil
	} else if strings.HasPrefix(path, z.apisDir) {
		return EventSrcAPI, z.apisDir, nil
	} else if strings.HasPrefix(path, z.routingsDir) {
		return EventSrcRouting, z.routingsDir, nil
	} else {
		log.Infof("watch event path is error: %s", path)
		return EventError, "", errors.Trace(ErrClosedClient)
	}
}

func (z *ZkStore) reWatch(path string) {
	switch {
	case path == "", path == "/":
		log.Errorf("path is null can not reWatch")
		return
	case path == z.clustersDir, path == z.serversDir, path == z.bindsDir, path == z.apisDir, path == z.routingsDir:
		z.cli.WatchC(path)
	default:
		z.cli.WatchData(path)
	}
}

func (z *ZkStore) setNumCByPath(path string, numC int32) (bool, error) {
	var flag bool = false
	if strings.HasPrefix(path, z.clustersDir) {
		log.Debugf("numC: <%d>", numC)
		log.Debugf("z.clustersDirCNum: <%d>", z.clustersDirCNum)
		if numC > z.clustersDirCNum {
			flag = true
		}
		z.clustersDirCNum = numC
		return flag, nil
	} else if strings.HasPrefix(path, z.serversDir) {
		log.Debugf("numC: <%d>", numC)
		log.Debugf("z.serversDirCNum: <%d>", z.serversDirCNum)
		if numC > z.serversDirCNum {
			flag = true
		}
		z.serversDirCNum = numC
		return flag, nil
	} else if strings.HasPrefix(path, z.bindsDir) {
		log.Debugf("numC: <%d>", numC)
		log.Debugf("z.bindsDirCNum: <%d>", z.bindsDirCNum)
		if numC > z.bindsDirCNum {
			flag = true
		}
		z.bindsDirCNum = numC
		return flag, nil
	} else if strings.HasPrefix(path, z.apisDir) {
		log.Debugf("numC: <%d>", numC)
		log.Debugf("z.apisDirCNum: <%d>", z.apisDirCNum)
		if numC > z.apisDirCNum {
			flag = true
		}
		z.apisDirCNum = numC
		return flag, nil
	} else if strings.HasPrefix(path, z.routingsDir) {
		log.Infof("numC: <%d>", numC)
		log.Infof("z.routingsDirCNum: <%d>", z.routingsDirCNum)
		if numC > z.routingsDirCNum {
			flag = true
		}
		z.routingsDirCNum = numC
		return flag, nil
	} else {
		return flag, nil
	}
}

func (z *ZkStore) doWatchWithCluster(evtType EvtType, path string) *Evt {
	if evtType == EventTypeReload {
		return &Evt{
			Src:   EventSrcCluster,
			Type:  evtType,
			Key:   path,
			Value: "",
		}
	}
	if evtType == EventTypeDelete {
		key := strings.TrimLeft(path, z.clustersDir)
		cluster := UnMarshalDelCluster(path, key)
		return &Evt{
			Src:   EventSrcCluster,
			Type:  evtType,
			Key:   key,
			Value: cluster,
		}
	}

	rs, err := z.cli.Read(path, false)

	if nil != err {
		log.InfoError(err)
		return nil
	}
	if nil == rs || len(rs) == 0 {
		return nil
	}
	cluster, marshalerr := UnMarshalCluster(rs)
	if marshalerr != nil {
		log.Errorf("doWatchWithCluster marshal is error: %s", string(rs))
		return nil
	}

	return &Evt{
		Src:   EventSrcCluster,
		Type:  evtType,
		Key:   path,
		Value: cluster,
	}
}

func (z *ZkStore) doWatchWithServer(evtType EvtType, path string) *Evt {
	if evtType == EventTypeReload {
		return &Evt{
			Src:   EventSrcServer,
			Type:  evtType,
			Key:   path,
			Value: "",
		}
	}

	if evtType == EventTypeDelete {
		addr := strings.TrimLeft(path, z.serversDir)
		server := UnMarshalDelServer(path, addr)
		return &Evt{
			Src:   EventSrcServer,
			Type:  evtType,
			Key:   addr,
			Value: server,
		}
	}

	rs, err := z.cli.Read(path, false)

	if nil != err {
		log.InfoError(err)
		return nil
	}
	if nil == rs || len(rs) == 0 {
		return nil
	}
	server, marshalerr := UnMarshalServer(rs)

	if marshalerr != nil {
		log.Errorf("doWatchWithServer marshal is error: %s", string(rs))
		return nil
	}

	return &Evt{
		Src:   EventSrcServer,
		Type:  evtType,
		Key:   path,
		Value: server,
	}
}

func (z *ZkStore) doWatchWithBind(evtType EvtType, path string) *Evt {
	if evtType == EventTypeReload {
		return &Evt{
			Src:   EventSrcBind,
			Type:  evtType,
			Key:   path,
			Value: "",
		}
	}
	infos := strings.SplitN(path, "-", 2)

	return &Evt{
		Src:  EventSrcBind,
		Type: evtType,
		Key:  path,
		Value: &Bind{
			ServerAddr:  strings.TrimLeft(infos[0], z.bindsDir+"/"),
			ClusterName: infos[1],
		},
	}
}

func (z *ZkStore) doWatchWithAPI(evtType EvtType, path string) *Evt {
	if evtType == EventTypeReload {
		return &Evt{
			Src:   EventSrcAPI,
			Type:  evtType,
			Key:   path,
			Value: "",
		}
	}

	if evtType == EventTypeDelete {
		url := strings.TrimLeft(path, z.apisDir)
		api := UnMarshalDelAPI(path, url)
		return &Evt{
			Src:   EventSrcAPI,
			Type:  evtType,
			Key:   path,
			Value: api,
		}
	}

	rs, err := z.cli.Read(path, false)

	if nil != err {
		log.InfoError(err)
		return nil
	}
	if nil == rs || len(rs) == 0 {
		return nil
	}
	api, marshalerr := UnMarshalAPI(rs)
	if marshalerr != nil {
		log.Errorf("doWatchWithAPI marshal is error: %s", string(rs))
		return nil
	}

	return &Evt{
		Src:   EventSrcAPI,
		Type:  evtType,
		Key:   path,
		Value: api,
	}
}

func (z *ZkStore) doWatchWithRouting(evtType EvtType, path string) *Evt {
	if evtType == EventTypeReload {
		return &Evt{
			Src:   EventSrcRouting,
			Type:  evtType,
			Key:   path,
			Value: "",
		}
	}

	if evtType == EventTypeDelete {
		id := strings.TrimLeft(path, z.routingsDir)
		routing := UnMarshalDelRouting(path, id)
		return &Evt{
			Src:   EventSrcRouting,
			Type:  evtType,
			Key:   path,
			Value: routing,
		}
	}

	rs, err := z.cli.Read(path, false)

	if nil != err {
		log.InfoError(err)
		return nil
	}
	if nil == rs || len(rs) == 0 {
		return nil
	}
	routing, marshalerr := UnMarshalRouting(rs)
	if marshalerr != nil {
		log.Errorf("doWatchWithRouting marshal is error: %s", string(rs))
	}

	return &Evt{
		Src:   EventSrcRouting,
		Type:  evtType,
		Key:   path,
		Value: routing,
	}
}

func (z *ZkStore) Close() {
	if !z.closed {
		z.cli.Close()
		z.closed = true
	}
}

func (z *ZkStore) init() {
	z.watchMethodMapping[EventSrcBind] = z.doWatchWithBind
	z.watchMethodMapping[EventSrcServer] = z.doWatchWithServer
	z.watchMethodMapping[EventSrcCluster] = z.doWatchWithCluster
	z.watchMethodMapping[EventSrcAPI] = z.doWatchWithAPI
	z.watchMethodMapping[EventSrcRouting] = z.doWatchWithRouting
	z.cli.Mkdir(z.prefix)
	z.cli.Mkdir(z.clustersDir)
	z.cli.Mkdir(z.serversDir)
	z.cli.Mkdir(z.bindsDir)
	z.cli.Mkdir(z.apisDir)
	z.cli.Mkdir(z.proxiesDir)
	z.cli.Mkdir(z.routingsDir)
	z.cli.Mkdir(z.deleteServersDir)
	z.cli.Mkdir(z.deleteClustersDir)
	z.cli.Mkdir(z.deleteAPIsDir)
}
