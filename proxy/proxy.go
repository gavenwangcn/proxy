package proxy

import (
	"container/list"
	"errors"
	"net/http"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/CodisLabs/codis/pkg/utils/log"
	l4g "github.com/alecthomas/log4go"
	"github.com/hotwheels/gateway/conf"
	"github.com/hotwheels/gateway/pkg/model"
	"github.com/valyala/fasthttp"
)

var (
	// ErrPrefixRequestCancel user cancel request error
	ErrPrefixRequestCancel = "request canceled"
	ErrDialConns           = "Dial Server Connect is error"
	// ErrNoServer no server
	ErrNoServer = errors.New("has no server")
	// ErrRewriteNotMatch rewrite not match request url
	ErrRewriteNotMatch = errors.New("rewrite not match request url")
)

var (
	// MergeContentType merge operation using content-type
	MergeContentType = "application/json; charset=utf-8"
	// MergeRemoveHeaders merge operation need to remove headers
	MergeRemoveHeaders = []string{
		"Content-Length",
		"Content-Type",
		"Date",
	}
)

// Proxy Proxy
type Proxy struct {
	clientPool    *ClientPool
	config        *conf.Conf
	routeTable    *model.RouteTable
	flushInterval time.Duration
	filters       *list.List
}

// NewProxy create a new proxy
func NewProxy(config *conf.Conf, routeTable *model.RouteTable) *Proxy {
	p := &Proxy{
		clientPool: NewClientPool(config),
		config:     config,
		routeTable: routeTable,
		filters:    list.New(),
	}

	return p
}

// RegistryFilter registry a filter
func (p *Proxy) RegistryFilter(name string) {
	f, err := newFilter(name, p.config, p)
	if nil != err {
		log.Panicf("Proxy unknow filter <%s>.", name)
	}

	p.filters.PushBack(f)
}

// Start start proxy
func (p *Proxy) Start() {
	err := p.startRPCServer()

	if nil != err {
		log.PanicErrorf(err, "Proxy start rpc at <%s> fail.", p.config.MgrAddr)
	}

	fastServer := &fasthttp.Server{
		Handler:         p.ReverseProxyHandler,
		ReadBufferSize:  p.config.ReadBufferSize,
		WriteBufferSize: p.config.WriteBufferSize,
		Logger:          log.StdLog,
	}

	log.ErrorErrorf(fastServer.ListenAndServe(p.config.Addr), "Proxy exit at %s", p.config.Addr)

	//log.ErrorErrorf(fasthttp.ListenAndServe(p.config.Addr, p.ReverseProxyHandler), "Proxy exit at %s", p.config.Addr)
}

// ReverseProxyHandler http reverse handler
func (p *Proxy) ReverseProxyHandler(ctx *fasthttp.RequestCtx) {
	//处理异常防止应用挂掉
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(error); !ok {
				log.InfoErrorf(r.(error), "PANIC: pkg: %v %s \n", debug.Stack())
			}
			ctx.Response.Reset()
			ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		}
	}()

	tStart := time.Now().UnixNano()

	results := p.routeTable.Select(&ctx.Request)
	if nil == results || len(results) == 0 {
		log.Debugf("404 url ip:<%s> :x-forwarded:<%s> :uri:<%s>", ctx.RemoteIP(), ctx.Request.Header.Peek("X-Forwarded-For"), ctx.Request.RequestURI())
		ctx.Response.Reset()
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		return
	}

	count := len(results)

	merge := count > 1

	c := &filterContext{}
	// Api merge
	if merge {
		wg := &sync.WaitGroup{}
		wg.Add(count)

		for _, result := range results {
			result.Merge = merge

			go func(result *model.RouteResult) {
				c = p.doProxy(ctx, wg, result)
			}(result)
		}

		wg.Wait()
	} else {
		//record upstream server cost
		c = p.doProxy(ctx, nil, results[0])
	}

	tEnd := time.Now().UnixNano()
	totalCost := tEnd - tStart
	log.Info(" Response start")
	for _, result := range results {
		if result.Err != nil || result.Code >= fasthttp.StatusInternalServerError {
			if result.API.Mock != nil {
				result.API.RenderMock(ctx)
				result.Release()
				p.recordAccsessLog(ctx, c, totalCost)
				return
			}
			result.Release()
			ctx.SetStatusCode(result.Code)
			return
		}

		if !merge {
			p.writeResult(ctx, result.Res)
			result.Release()
			p.recordAccsessLog(ctx, c, totalCost)
			return
		}
	}

	for _, result := range results {
		for _, h := range MergeRemoveHeaders {
			result.Res.Header.Del(h)
		}
		result.Res.Header.CopyTo(&ctx.Response.Header)
	}
	log.Info(" WriteResponse start")
	ctx.Response.Header.SetContentType(MergeContentType)
	ctx.SetStatusCode(fasthttp.StatusOK)

	ctx.WriteString("{")

	for index, result := range results {
		ctx.WriteString("\"")
		ctx.WriteString(result.Node.AttrName)
		ctx.WriteString("\":")
		ctx.Write(result.Res.Body())
		if index < count-1 {
			ctx.WriteString(",")
		}

		result.Release()
	}

	ctx.WriteString("}")
	log.Info(" WriteResponse end")

	log.Info(" Response end")

	p.recordAccsessLog(ctx, c, totalCost)
}

//return server cost
func (p *Proxy) doProxy(ctx *fasthttp.RequestCtx, wg *sync.WaitGroup, result *model.RouteResult) *filterContext {
	if nil != wg {
		defer wg.Done()
	}

	svr := result.Svr

	if nil == svr {
		result.Err = ErrNoServer
		result.Code = http.StatusServiceUnavailable
		return nil
	}
	outreq := copyRequest(&ctx.Request)

	// change url
	if result.NeedRewrite() {
		// if not use rewrite, it only change uri path and query string
		realPath := result.GetRewritePath(&ctx.Request)
		if "" != realPath {
			log.Debugf("URL Rewrite from <%s> to <%s> ", string(ctx.URI().FullURI()), realPath)
			outreq.SetRequestURI(realPath)
		} else {
			log.Warnf("URL Rewrite<%s> not matches <%s>", string(ctx.URI().FullURI()), result.Node.Rewrite)
			result.Err = ErrRewriteNotMatch
			result.Code = http.StatusBadRequest
			return nil
		}
	}
	//设置host
	host := result.API.Host
	if "" != host {
		outreq.SetHost(host)
	}
	c := &filterContext{
		ctx:        ctx,
		outreq:     outreq,
		result:     result,
		rb:         p.routeTable,
		runtimeVar: make(map[string]string),
	}

	// pre filters
	filterName, code, err := p.doPreFilters(c)
	if nil != err {
		log.WarnErrorf(err, "Proxy Filter-Pre<%s> fail", filterName)
		result.Err = err
		result.Code = code
		return nil
	}
	c.startAt = time.Now().UnixNano()
	fastHTTPClient := p.clientPool.GetDefaultClient(svr.Addr)
	res, err := fastHTTPClient.Do(outreq)
	c.endAt = time.Now().UnixNano()

	result.Res = res

	if err != nil || res.StatusCode() >= fasthttp.StatusInternalServerError {
		resCode := res.StatusCode()
		if nil != err {
			log.InfoErrorf(err, "Proxy Fail <%s>", svr.Addr)
			//被动检活 如果后端服务无法创建连接 认为检活失败（默认超时时间3秒），默认检活次数3次
			if strings.HasPrefix(err.Error(), ErrDialConns) {
				svr.Fail()
				if svr.CheckFailCount > 2 {
					p.routeTable.DownServer(svr)
					log.Errorf("Server Can not Connect, DownServer <%s>", svr.Addr)
				}
			}
		} else {
			resCode = res.StatusCode()
			log.InfoErrorf(err, "Proxy Fail <%s>, Code <%d>", svr.Addr, res.StatusCode())
		}

		// 用户取消，不计算为错误
		if nil == err || !strings.HasPrefix(err.Error(), ErrPrefixRequestCancel) {
			p.doPostErrFilters(c)
		}

		result.Err = err
		result.Code = resCode
		return nil
	}

	//log.Infof("Backend server[%s] responsed, code <%d>, body<%s>", svr.Addr, res.StatusCode(), res.Body())

	// post filters
	filterName, code, err = p.doPostFilters(c)
	if nil != err {
		log.InfoErrorf(err, "Proxy Filter-Post<%s> fail: %s ", filterName, err.Error())

		result.Err = err
		result.Code = code
		return nil
	}
	return c
}

func (p *Proxy) writeResult(ctx *fasthttp.RequestCtx, res *fasthttp.Response) {
	ctx.SetStatusCode(res.StatusCode())
	ctx.Write(res.Body())
}

// access log added by 15070229ß
// log format: '$remote_addr\t$http_x_forwarded_for\t-\t
// 		$remote_user\t$time_iso8601\t$request_method\t"$document_uri"\t
//		"$query_string"\t$server_protocol\t$status\t$body_bytes_sent\t
//		$request_time\t"$http_referer"\t"$http_user_agent"\t$http_Cdn-Src-Ip\t
//		$host\t$server_time\t$server_addr\t-\t-\t-\t-\t-\tV5';
func (p *Proxy) recordAccsessLog(ctx *fasthttp.RequestCtx, c *filterContext, totalCost int64) {
	// Don't exit on panic
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(error); !ok {
				log.InfoErrorf(r.(error), "PANIC: pkg: %v %s \n", debug.Stack())
			}
		}
	}()

	if c == nil || ctx == nil {
		return
	}
	l4g.Info("%s\t%s\t-\t%s\t%s\t%s\t\"%s\"\t\"%s\"\t%s\t%d\t%d\t%d\t\"%s\"\t\"%s\"\t%s\t%s\t%d\t%s\t-\t-\t-\t-\t-\tV5",
		ctx.RemoteIP(),
		ctx.Request.Header.Peek("X-Forwarded-For"),
		ctx.RemoteAddr(),
		time.Now().String(),
		ctx.Method(),
		ctx.Request.URI().FullURI(),
		ctx.Request.URI().QueryString(),
		ctx.Request.URI().Scheme(),
		ctx.Response.StatusCode(),
		ctx.Response.Header.ContentLength(),
		totalCost/1000,
		ctx.Request.Header.Referer(),
		ctx.Request.Header.UserAgent(),
		ctx.Request.Header.Peek("Cdn-Src-Ip"),
		ctx.Request.Host(),
		(c.endAt-c.startAt)/1000,
		c.result.Svr.Addr)
}
