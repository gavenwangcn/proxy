package proxy

import (
	"net"
	"time"

	"github.com/CodisLabs/codis/pkg/utils/errors"
	"github.com/hotwheels/gateway/conf"
	"github.com/valyala/fasthttp"
)

// fast http client
type DefaultClient struct {
	client *fasthttp.HostClient
	Addr   string
}

// NewFastHTTPClient create FastHTTPClient instance
func NewDefaultClient(conf *conf.Conf, addr string) *DefaultClient {
	client := &fasthttp.HostClient{
		Addr:                addr,
		MaxIdleConnDuration: time.Duration(conf.MaxIdleConnDuration) * time.Second,
		ReadTimeout:         time.Duration(conf.ReadTimeout) * time.Second,
		WriteTimeout:        time.Duration(conf.WriteTimeout) * time.Second,
		MaxResponseBodySize: conf.MaxResponseBodySize,
		ReadBufferSize:      conf.ReadBufferSize,
		WriteBufferSize:     conf.WriteBufferSize,
		MaxConns:            conf.MaxConns,
		Dial:                dial,
	}
	if conf.MaxConnDuration > 0{
		client.MaxConnDuration = time.Duration(conf.MaxConnDuration) * time.Second
	}
	return &DefaultClient{
		client: client,
		Addr:   addr,
	}
}

// Do do proxy
func (c *DefaultClient) Do(req *fasthttp.Request) (*fasthttp.Response, error) {
	resp := fasthttp.AcquireResponse()
	err := c.client.Do(req, resp)
	return resp, err
}

func dial(addr string) (net.Conn, error) {
	conn, err := fasthttp.Dial(addr)
	if err != nil {
		return nil, errors.Errorf("Dial Server Connect is error %s", err.Error())
	}
	if conn == nil {
		return nil, errors.Errorf("Diall Error Connect is nil %s", err.Error())
	}

	return conn, nil
}
