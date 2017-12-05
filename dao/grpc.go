package dao

import (
	"context"
	"fmt"
	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/opentracing/opentracing-go"
	"github.com/tonyjt/tgo_v2/config"
	"github.com/tonyjt/tgo_v2/log"
	"github.com/tonyjt/tgo_v2/pconst"
	"github.com/tonyjt/tgo_v2/terror"
	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer"
	//"google.golang.org/grpc/resolver"
	"github.com/opentracing/opentracing-go/ext"
	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/resolver/manual"
	"sync"
)

type Grpc struct {
	Service     string
	Insecure    bool
	DialOptions []grpc.DialOption
}

var (
	grpcConnMap map[string]*grpc.ClientConn
	grpcConnMux sync.RWMutex
)

func (p *Grpc) ZipkinNewSpan(ctx context.Context, name string) (opentracing.Span, context.Context) {
	if config.FeatureZipkin() {
		return opentracing.StartSpanFromContext(ctx, fmt.Sprintf("grpc:%s:%s", p.Service, name))
	} else {
		return nil, ctx
	}
}
func (p *Grpc) proccessError(span opentracing.Span, err error, msg string) error {
	log.Error(msg)
	if span != nil {
		ext.Error.Set(span, true)
		span.SetTag("err", err)
	}
	return err
}

func (p *Grpc) GetConn(ctx context.Context) (conn *grpc.ClientConn, err error) {

	span, ctx := p.ZipkinNewSpan(ctx, "getconn")

	if span != nil {
		defer span.Finish()
	}

	grpcConnMux.Lock()
	defer grpcConnMux.Unlock()
	if grpcConnMap == nil {
		grpcConnMap = make(map[string]*grpc.ClientConn)
	}

	conn, ok := grpcConnMap[p.Service]

	if !ok || conn == nil {
		conf := config.GrpcGet(p.Service)
		if conf == nil {
			err = terror.New(pconst.ERROR_GRPC_CONFIG)
			return
		}
		b := balancer.Get("round_robin")

		dialOptions := append(p.DialOptions, grpc.WithBalancerBuilder(b))
		if conf.Insecure {
			dialOptions = append(dialOptions, grpc.WithInsecure())
		}
		if config.FeatureZipkin() {
			tracer := opentracing.GlobalTracer()
			dialOptions = append(dialOptions, grpc.WithUnaryInterceptor(otgrpc.OpenTracingClientInterceptor(tracer)))
		}
		r, cleanup := manual.GenerateAndRegisterManualResolver()
		defer cleanup()

		conn, err = grpc.Dial(r.Scheme()+":///test.server", dialOptions...)

		if err != nil {
			msg := fmt.Sprintf("dail failed,service:%s,error:%s", p.Service, err.Error())
			err = terror.New(pconst.ERROR_GRPC_DAIL)
			p.proccessError(span, err, msg)
			return
		}
		var addr []resolver.Address
		for _, a := range conf.Conn {
			addr = append(addr, resolver.Address{Addr: a})
		}
		r.NewAddress(addr)

		grpcConnMap[p.Service] = conn
	}

	return

}

func (p *Grpc) CloseConn(ctx context.Context, conn *grpc.ClientConn) error {

	span, ctx := p.ZipkinNewSpan(ctx, "colseconn")

	if span != nil {
		defer span.Finish()
	}

	/*grpcPool, ok := grpcPoolMap[p.ServerName]

	if !ok {
		return errors.New("grpc pool is not exist")
	}*/
	return nil //grpcPool.ReturnObject(conn)
}

func (p *Grpc) Invoke(ctx context.Context, conn *grpc.ClientConn, funcName string,
	funcInvoke func(context.Context) error) (err error) {

	span, ctx := p.ZipkinNewSpan(ctx, fmt.Sprintf("invoke:%s", funcName))

	if span != nil {
		defer span.Finish()
	}

	if conn != nil {
		defer p.CloseConn(ctx, conn)
	}

	err = funcInvoke(ctx)

	if err != nil {
		msg := fmt.Sprintf("grpc error :%s", err.Error())
		err = terror.New(pconst.ERROR_GRPC_INVOKE)
		p.proccessError(span, err, msg)
	}

	return

}
