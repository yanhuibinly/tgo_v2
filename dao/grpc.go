package dao

import (
	"github.com/tonyjt/gogrpc"
	"google.golang.org/grpc"
	"github.com/tonyjt/tgo_v2/config"
	"github.com/tonyjt/tgo_v2/terror"
	"github.com/tonyjt/tgo_v2/pconst"
	"github.com/tonyjt/tgo_v2/log"
	"context"
	"sync"
	"fmt"
	"github.com/opentracing/opentracing-go"
)

type Grpc struct{
	Service string
	DialOptions []grpc.DialOption
}

var (
	grpcConnMap map[string]*grpc.ClientConn
	grpcConnMux sync.RWMutex
)

func (p *Grpc) ZipkinNewSpan(ctx context.Context,name string)(opentracing.Span,context.Context){
	if config.FeatureZipkin(){
		return opentracing.StartSpanFromContext(ctx,fmt.Sprintf("grpc:%s",name))
	}else{
		return nil,ctx
	}
}
func (p *Grpc) proccessError(span opentracing.Span,err error,msg string)(error){
	log.Error(msg)
	if span !=nil{
		span.SetTag("err",err)
	}
	return err
}

func (p *Grpc) GetConn(ctx context.Context) (conn *grpc.ClientConn,err error) {

	span,ctx:= p.ZipkinNewSpan(ctx,"getconn")

	if span !=nil{
		defer span.Finish()
	}

	grpcConnMux.Lock()
	defer grpcConnMux.Unlock()
	if grpcConnMap == nil {
		grpcConnMap = make(map[string]*grpc.ClientConn)
	}

	conn, ok := grpcConnMap[p.Service]

	if !ok || conn == nil {
		config := config.GrpcGet(p.Service)
		if config == nil {
			err = terror.New(pconst.ERROR_GRPC_CONFIG)
			return
		}
		balancer := gogrpc.NewBalancerIp()
		balancer.SetAddr(config.Conn...)

		dialOptions := append(p.DialOptions, grpc.WithBalancer(balancer))
		conn, err = grpc.Dial(p.Service, dialOptions...)

		if err != nil {
			msg:=fmt.Sprintf("dail failed,service:%s,error:%s",p.Service,err.Error())
			err = terror.New(pconst.ERROR_GRPC_DAIL)
			p.proccessError(span,err,msg)
			return 
		}
		grpcConnMap[p.Service] = conn
	}

	return

}

func (p *Grpc) CloseConn(ctx context.Context,conn *grpc.ClientConn) error {

	span,ctx:= p.ZipkinNewSpan(ctx,"colseconn")

	if span !=nil{
		defer span.Finish()
	}

	/*grpcPool, ok := grpcPoolMap[p.ServerName]

	if !ok {
		return errors.New("grpc pool is not exist")
	}*/
	return nil //grpcPool.ReturnObject(conn)
}

func (p *Grpc) Invoke(ctx context.Context,conn *grpc.ClientConn,
	funcInvoke func(context.Context,interface{}, ...grpc.CallOption)(interface{},error),in interface{},opts ...grpc.CallOption,
	)(reply interface{},err error){
	span,ctx:= p.ZipkinNewSpan(ctx,"invoke")

	if span !=nil{
		defer span.Finish()
	}

	if conn !=nil{
		defer p.CloseConn(ctx,conn)
	}

	reply,err = funcInvoke(ctx,in,opts...)

	return

}