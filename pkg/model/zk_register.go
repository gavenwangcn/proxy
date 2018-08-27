package model

import (
	"fmt"
	"net"
	"time"

	"github.com/CodisLabs/codis/pkg/utils/log"
	"github.com/hotwheels/gateway/pkg/util"
	"github.com/samuel/go-zookeeper/zk"
	tk "github.com/toolkits/net"
)

// Registry registry self
func (z ZkStore) Registry(proxyInfo *ProxyInfo) error {
	z.doRegistry(proxyInfo)
	return nil
}

func (z ZkStore) doRegistry(proxyInfo *ProxyInfo) {
	proxyInfo.Conf.Addr = convertIP(proxyInfo.Conf.Addr)
	proxyInfo.Conf.MgrAddr = convertIP(proxyInfo.Conf.MgrAddr)
	proxyInfo.Key = getInternal()
	z.proxyInfo = proxyInfo
	key := fmt.Sprintf("%s/%s", z.proxiesDir, proxyInfo.Key)
	e, err := z.cli.CreateEphemeral(key, proxyInfo.MarshalB())
	if err != nil {
		log.ErrorError(err, "Registry fail.")
	}
	z.watchProxy = e
	go z.doWatchProxy()
}

func (z *ZkStore) doWatchProxy() {
	for {
		select {
		case change := <-z.watchProxy:
			log.Debugf("watchProxy Event: <%s>", change.Type)
			err := z.dispathProxyEvent(change)
			if err != nil {
				log.Errorf("watchProxy Event error <%s", err.Error())
			}
		}
	}
}

func (z *ZkStore) dispathProxyEvent(change zk.Event) error {
	path := change.Path
	if change.Type == zk.EventNotWatching {
		log.Infof("zk reconnect ok for ReRegistryProxy")
		e, err := z.cli.CreateEphemeral(path, z.proxyInfo.MarshalB())
		if err != nil {
			log.ErrorError(err, "ReRegistryProxy fail.")
		}
		oldProxyE := z.watchProxy
		z.watchProxy = e
		close(oldProxyE)
		return nil
	} else if change.Type == zk.EventNodeDataChanged {
		rs, err := z.cli.Read(path, false)

		if nil != err {
			log.InfoError(err)
			return nil
		}
		if nil == rs || len(rs) == 0 {
			return nil
		}
		proxyInfo, marshalerr := UnMarshalProxyInfo(rs)
		if marshalerr != nil {
			log.Errorf("doWatchProxyEvent marshal is error: %s", string(rs))
			return nil
		}
		//设置proxy log level
		z.proxyInfo = proxyInfo
		level := util.SetLogLevel(proxyInfo.Conf.LogLevel)
		log.Infof("change log level: <%s>", level)
		e, errw := z.cli.Watch(path)
		if errw != nil {
			log.ErrorErrorf(errw, "zk watchData error: <%s>", path)
		}
		oldProxyE := z.watchProxy
		z.watchProxy = e
		close(oldProxyE)
	} else {
		log.Debugf("zkclient dispathProxyEvent not Metched, Type <d%>", change.Type)
		return nil
	}

	log.Infof("zk Proxy changed: <%s>", path)

	return nil

}

func getInternal() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.ErrorError(err, "Get internal ip.")
		return "error ip"
	}
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return "0.0.0.0"
}

// GetProxies return runable proxies
func (z ZkStore) GetProxies() ([]*ProxyInfo, error) {

	_, ls, err := z.cli.List(z.proxiesDir, true)

	if nil != err {
		return nil, err
	}

	l := len(ls)
	proxies := make([]*ProxyInfo, l)

	for i := 0; i < l; i++ {
		rs, _ := z.cli.Read(ls[i], true)
		v, marshalerr := UnMarshalProxyInfo(rs)
		if marshalerr != nil {
			log.Errorf("marshal routing value is error %s", string(rs))
			continue
		}
		proxies[i] = v
	}

	if nil != err {
		return nil, err
	}

	return proxies, nil
}

// ChangeLogLevel change proxy log level
func (z ZkStore) ChangeLogLevel(addr string, level string) error {
	rpcClient, _ := tk.JsonRpcClient("tcp", addr, time.Second*5)

	req := SetLogReq{
		Level: level,
	}

	rsp := &SetLogRsp{
		Code: 0,
	}

	return rpcClient.Call("Manager.SetLogLevel", req, rsp)
}

// AddAnalysisPoint add a analysis point
func (z ZkStore) AddAnalysisPoint(proxyAddr, serverAddr string, secs int) error {
	rpcClient, _ := tk.JsonRpcClient("tcp", proxyAddr, time.Second*5)

	req := AddAnalysisPointReq{
		Addr: serverAddr,
		Secs: secs,
	}

	rsp := &AddAnalysisPointRsp{
		Code: 0,
	}

	return rpcClient.Call("Manager.AddAnalysisPoint", req, rsp)
}

// GetAnalysisPoint return analysis point data
func (z ZkStore) GetAnalysisPoint(proxyAddr, serverAddr string, secs int) (*GetAnalysisPointRsp, error) {
	rpcClient, err := tk.JsonRpcClient("tcp", proxyAddr, time.Second*5)

	if nil != err {
		return nil, err
	}

	req := GetAnalysisPointReq{
		Addr: serverAddr,
		Secs: secs,
	}

	rsp := &GetAnalysisPointRsp{}

	err = rpcClient.Call("Manager.GetAnalysisPoint", req, rsp)

	return rsp, err
}
