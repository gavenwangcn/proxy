package conf

// Conf config struct
type Conf struct {
	LogLevel string `json:"logLevel"`

	Addr      string   `json:"addr"`
	MgrAddr   string   `json:"mgrAddr"`
	ConfType  string   `json:"confType"`
	EtcdAddrs []string `json:"etcdAddrs"`
	ZkAddrs   string   `json:"zkAddrs"`
	Prefix    string   `json:"prefix"`

	Filers []string `json:"filers"`

	// Maximum number of connections which may be established to server
	MaxConns int `json:"maxConns"`

	MaxConnDuration int `json:"maxConnDuration,0"`
	// MaxIdleConnDuration Idle keep-alive connections are closed after this duration.
	MaxIdleConnDuration int `json:"maxIdleConnDuration"`
	// ReadBufferSize Per-connection buffer size for responses' reading.
	ReadBufferSize int `json:"readBufferSize"`
	// WriteBufferSize Per-connection buffer size for requests' writing.
	WriteBufferSize int `json:"writeBufferSize"`
	// ReadTimeout Maximum duration for full response reading (including body).
	ReadTimeout int `json:"readTimeout"`
	// WriteTimeout Maximum duration for full request writing (including body).
	WriteTimeout int `json:"writeTimeout"`
	// Flush Client Tcp Connect time.
	FlushDuration int `json:"flushDuration"`
	// MaxResponseBodySize Maximum response body size.
	MaxResponseBodySize int `json:"maxResponseBodySize"`

	// EnableAdminCheck enable pprof
	EnableProxyCheck bool `json:"enableProxyCheck"`

	// EnablePPROF enable pprof
	EnablePPROF bool `json:"enablePPROF"`
	// PPROFAddr pprof addr
	PPROFAddr string `json:"pprofAddr,omitempty"`
}
