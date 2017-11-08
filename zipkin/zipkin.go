package zipkin

import (
	"github.com/openzipkin/zipkin-go-opentracing"
	"github.com/opentracing/opentracing-go"
	"github.com/tonyjt/tgo_v2/config"
	"github.com/tonyjt/tgo_v2/log"
	"fmt"
	"net"
)

func Load(hostPort string){
	config := config.ZipkinGet()

	collector,err:= zipkintracer.NewHTTPCollector(config.CollectorEndpoint)

	if err!=nil{
		log.Errorf("unable to create zipkin http collector : %+v",err)

		panic(err)
	}

	var host string
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		panic(err)
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				host = ipnet.IP.String()
			}
		}
	}

	recorder:= zipkintracer.NewRecorder(collector,config.Debug,fmt.Sprintf("%s%s",host, hostPort),config.ServiceName)

	tracer,err:= zipkintracer.NewTracer(
		recorder,
		zipkintracer.ClientServerSameSpan(config.SameSpan),
		zipkintracer.TraceID128Bit(config.TraceID128Bit),
	)

	if err!=nil{
		log.Errorf("unable to create Zipkin tracer: %+v", err)
		panic(err)
	}

	opentracing.InitGlobalTracer(tracer)
}
