// @15070229 filter_cookie
package proxy

import (
	"github.com/hotwheels/gateway/conf"
)

// CookiesFilter record the http access log
// log format: $remoteip "$method $path" $code "$cookies"
type CustomHeadersFilter struct {
	baseFilter
	config *conf.Conf
	proxy  *Proxy
}

func newCustomHeadersFilter(config *conf.Conf, proxy *Proxy) Filter {
	return CustomHeadersFilter{
		config: config,
		proxy:  proxy,
	}
}

// Name return name of this filter
func (f CustomHeadersFilter) Name() string {
	return FilterCustomHeader
}

// Pre execute before proxy
func (f CustomHeadersFilter) Pre(c *filterContext) (statusCode int, err error) {

	for _, h := range c.result.API.ReqHeaders {
		if "" != h.Name && "" != h.Value {
			c.outreq.Header.Add(h.Name, h.Value)
		}
	}

	return f.baseFilter.Pre(c)
}

// Post execute after proxy
func (f CustomHeadersFilter) Post(c *filterContext) (statusCode int, err error) {

	for _, h := range c.result.API.CustomHeaders {
		if "" != h.Name && "" != h.Value {
			c.result.Res.Header.Add(h.Name, h.Value)
		}
	}

	return f.baseFilter.Post(c)
}
