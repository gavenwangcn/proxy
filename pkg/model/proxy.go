package model

import (
	"encoding/json"

	"github.com/hotwheels/gateway/conf"
)

// ProxyInfo proxy info
type ProxyInfo struct {
	Key  string     `json:"key, omitempty"`
	Conf *conf.Conf `json:"conf,omitempty"`
}

// UnMarshalProxyInfo unmarshal
func UnMarshalProxyInfo(data []byte) (*ProxyInfo, error) {
	v := &ProxyInfo{}
	err := json.Unmarshal(data, v)
	if err != nil {
		return nil, err
	}
	return v, nil
}

// Marshal marshal
func (p *ProxyInfo) Marshal() string {
	v, _ := json.Marshal(p)
	return string(v)
}

// Marshal marshalB
func (p *ProxyInfo) MarshalB() []byte {
	v, _ := json.Marshal(p)
	return v
}
