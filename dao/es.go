package dao

import (
	"context"
	"fmt"
	"github.com/olivere/elastic"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/tonyjt/tgo_v2/config"
	"github.com/tonyjt/tgo_v2/log"
	"github.com/tonyjt/tgo_v2/pconst"
	"github.com/tonyjt/tgo_v2/terror"
)

type Es struct {
	Service string
	Index   string
	Type    string
}

func (p *Es) ZipkinNewSpan(ctx context.Context, name string) (opentracing.Span, context.Context) {
	if config.FeatureZipkin() {
		return opentracing.StartSpanFromContext(ctx, fmt.Sprintf("es:%s:%s", p.Service, name))
	} else {
		return nil, ctx
	}
}
func (p *Es) proccessError(span opentracing.Span, err error, msg string) error {
	log.Error(msg)
	if span != nil {
		ext.Error.Set(span, true)
		span.SetTag("err", err)
	}
	return err
}

func (p *Es) GetConn(ctx context.Context) (client *elastic.Client, err error) {
	cnf := config.EsGet(p.Service)
	client, err = elastic.NewClient(elastic.SetURL(cnf.Conn...))

	if err != nil {
		msg := fmt.Sprintf("new client error, service:%s,error:%s,conn:%v", p.Service, err.Error(), cnf.Conn)

		err = terror.New(pconst.ERROR_ES_CONFIG)

		var span opentracing.Span

		span, ctx = p.ZipkinNewSpan(ctx, "getconn")

		if span != nil {
			defer span.Finish()
		}

		p.proccessError(span, err, msg)
	}
	return
}

// Invoke
func (p *Es) Invoke(ctx context.Context, client *elastic.Client, funcName string,
	funcInvoke func(context.Context) error) (err error) {

	span, ctx := p.ZipkinNewSpan(ctx, fmt.Sprintf("invoke:%s", funcName))

	if span != nil {
		defer span.Finish()
	}

	err = funcInvoke(ctx)

	if err != nil {
		msg := fmt.Sprintf("es invoke error :%s", err.Error())
		err = terror.New(pconst.ERROR_ES_INVOKE)
		p.proccessError(span, err, msg)
	}

	return
}

func (p *Es) Insert(ctx context.Context, indexName string, id string, data interface{}) (err error) {

	span, ctx := p.ZipkinNewSpan(ctx, "insert")

	if span != nil {
		defer span.Finish()
	}

	client, err := p.GetConn(ctx)

	if err != nil {
		return
	}

	client.Index().Index(indexName)

	return
}
