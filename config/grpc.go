package config


type Grpc struct{
	Grpc []GrpcConf
}

type GrpcConf struct {
	Service      string
	Insecure bool
	Conn []string
}


var (
	grpcConfig map[string]*GrpcConf
)
func init() {

	if FeatureGrpc() {
		config := &Grpc{}

		defaultGrpcConfig := configGrpcGetDefault()

		configGet("grpc", config, defaultGrpcConfig)

		if len(config.Grpc) == 0 {
			panic("grpc config is empty")
		}

		grpcConfig = make(map[string]*GrpcConf)

		for i, c := range config.Grpc {
			grpcConfig[c.Service] = &config.Grpc[i]
		}
	}
}



func configGrpcGetDefault() *Grpc {
	return &Grpc{Grpc:[]GrpcConf{GrpcConf{Service:"tgo",Conn:[]string{}}}}
}

func GrpcGet(service string)(*GrpcConf){
	g,ok:= grpcConfig[service]

	if !ok{
		return nil
	}
	return g
}