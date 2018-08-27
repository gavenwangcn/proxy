package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"net/http"
	_ "net/http/pprof"
	"runtime"

	"github.com/CodisLabs/codis/pkg/utils/log"
	"github.com/hotwheels/gateway/conf"
	"github.com/hotwheels/gateway/pkg/model"
	"github.com/hotwheels/gateway/pkg/util"
	"github.com/hotwheels/gateway/proxy"
)

var (
	logFile       = flag.String("log-file", "", "which file to record error log, if not set stdout to use.")
	logLevel      = flag.String("log-level", "info", "log level.")
	accessLogFile = flag.String("access-log-file", "access.log", "which file to record access log,if not set stdout to use.")
	configFile    = flag.String("config", "config.json", "config file")
)

func main() {
	flag.Parse()

	runtime.GOMAXPROCS(runtime.NumCPU())

	util.InitLog(*logFile)
	util.InitAccessLog(*accessLogFile)
	level := util.SetLogLevel(*logLevel)

	data, err := ioutil.ReadFile(*configFile)
	if err != nil {
		log.PanicErrorf(err, "read config file <%s> failure.", *configFile)
	}

	cnf := &conf.Conf{}
	err = json.Unmarshal(data, cnf)
	if err != nil {
		log.PanicErrorf(err, "parse config file <%s> failure.", *configFile)
	}

	cnf.LogLevel = level

	if cnf.EnablePPROF {
		go func() {
			log.Println(http.ListenAndServe(cnf.PPROFAddr, nil))
		}()
	}

	log.Infof("conf:<%+v>", cnf)

	proxyInfo := &model.ProxyInfo{
		Conf: cnf,
	}

	store, storeErr := model.NewStore(cnf.ConfType, cnf.EtcdAddrs, cnf.ZkAddrs, cnf.Prefix)

	if storeErr != nil {
		log.Panicf("init store error:<%+v>", storeErr)
	}

	register, _ := store.(model.Register)

	if cnf.ConfType == "zk" ||
		cnf.ConfType == "etcd" {
		register.Registry(proxyInfo)
	}

	rt := model.NewRouteTable(store)
	rt.ChangeProxyCheck(cnf.EnableProxyCheck)
	rt.Load()

	server := proxy.NewProxy(cnf, rt)

	for _, filter := range cnf.Filers {
		server.RegistryFilter(filter)
	}

	go rt.FlushSvrsChecks(cnf.FlushDuration)
	server.Start()
}
