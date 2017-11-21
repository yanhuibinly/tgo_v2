package dao

import (
	"context"
	"fmt"
	"github.com/opentracing/opentracing-go"
	"github.com/tonyjt/gogrpc"
	"github.com/tonyjt/tgo_v2/config"
	"github.com/tonyjt/tgo_v2/log"
	"github.com/tonyjt/tgo_v2/pconst"
	"github.com/tonyjt/tgo_v2/terror"
	"google.golang.org/grpc"
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
		balancer := gogrpc.NewBalancerIp()

		balancer.SetAddr(conf.Conn...)

		dialOptions := append(p.DialOptions, grpc.WithBalancer(balancer))
		if conf.Insecure {
			dialOptions = append(dialOptions, grpc.WithInsecure())
		}
		conn, err = grpc.Dial(p.Service, dialOptions...)

		if err != nil {
			msg := fmt.Sprintf("dail failed,service:%s,error:%s", p.Service, err.Error())
			err = terror.New(pconst.ERROR_GRPC_DAIL)
			p.proccessError(span, err, msg)
			return
		}
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
		log.Errorf("grpc error :%s", err.Error())
		err = terror.New(pconst.ERROR_GRPC_INVOKE)
	}

	return

}
