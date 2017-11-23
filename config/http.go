package config

import (
	"time"
)

type Http struct {
	Http []HttpConf
}

type HttpConf struct {
	Service string
	Conn    HttpConn
	Paths   []HttpPath
}

type HttpConn struct {
	Url     string
	Timeout time.Duration
}

type HttpPath struct {
	Key  string
	Path string
}

var (
	httpConfig map[string]*HttpConf
)

func init() {
	if FeatureHttp() {
		conf := &Http{}

		defaultHttpConfig := configHttpGetDefault()

		configGet("http", conf, defaultHttpConfig)

		if len(conf.Http) == 0 {
			panic("http config is empty")
		}

		httpConfig = make(map[string]*HttpConf)

		for i, c := range conf.Http {
			httpConfig[c.Service] = &conf.Http[i]
		}
	}
	return
}

func configHttpGetDefault() *Http {
	return &Http{Http: []HttpConf{HttpConf{Service: "tgo", Conn: HttpConn{Url: ""}}}}
}

func HttpGet(service string) *HttpConf {

	if httpConfig == nil {
		panic("http config is nil")
	}
	g, ok := httpConfig[service]

	if !ok {
		return nil
	}
	return g
}

func HttpGetPath(conf *HttpConf, key string) string {
	if conf == nil {
		return ""
	}

	for _, p := range conf.Paths {
		if p.Key == key {
			return p.Path
		}
	}
	return ""
}
