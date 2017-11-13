package config

import "time"

type Http struct{
	Http []HttpConf
}

type HttpConf struct{
	Serivce string
	Conn HttpConn
}

type HttpConn struct {
	Url string
	Timeout time.Duration
}

var (
	httpConfig map[string]*HttpConf
)

func init(){
	if FeatureHttp() {
		config:= &Http{}

		defaultHttpConfig := configHttpGetDefault()

		configGet("http", config, defaultHttpConfig)

		if len(config.Http) ==0{
			panic("http config is empty")
		}

		httpConfig = make(map[string]*HttpConf)

		for _,c:= range config.Http{
			httpConfig[c.Serivce] = &c
		}
	}
	return
}

func configHttpGetDefault() *Http {
	return &Http{Http:[]HttpConf{HttpConf{Serivce:"tgo",Conn:HttpConn{Url:""}}}}
}

func HttpGet(service string)(*HttpConf){
	g,ok:= httpConfig[service]

	if !ok{
		return nil
	}
	return g
}