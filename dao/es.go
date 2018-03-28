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
	"reflect"
)

//Es es struct
type Es struct {
	Service string
	Index   string
	Type    string
}

//ZipkinNewSpan new zipkin span for es
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

func (p *Es) Insert(ctx context.Context, id string, data interface{}) (err error) {

	span, ctx := p.ZipkinNewSpan(ctx, "insert")
	if span != nil {
		defer span.Finish()
	}

	client, err := p.GetConn(ctx)
	if err != nil {
		return
	}

	_, err = client.Index().Index(p.Index).Type(p.Type).Id(id).BodyJson(data).Do(ctx)
	if err != nil {
		log.Errorf("es insert error :%s", err.Error())
	}

	return
}

func (p *Es) Update(ctx context.Context, id string, doc interface{}) (err error) {

	span, ctx := p.ZipkinNewSpan(ctx, "update")
	if span != nil {
		defer span.Finish()
	}

	client, err := p.GetConn(ctx)
	if err != nil {
		return
	}

	_, err = client.Update().Index(p.Index).Type(p.Type).Id(id).Doc(doc).Do(ctx)
	if err != nil {
		log.Errorf("es update error :%s", err.Error())
	}

	return
}

func (p *Es) Delete(ctx context.Context, id string) (err error) {

	span, ctx := p.ZipkinNewSpan(ctx, "delete")
	if span != nil {
		defer span.Finish()
	}

	client, err := p.GetConn(ctx)
	if err != nil {
		return
	}

	_, err = client.Delete().Index(p.Index).Type(p.Type).Id(id).Do(ctx)
	if err != nil {
		log.Errorf("es delete error :%s", err.Error())
	}

	return
}

func (p *Es) Search(ctx context.Context, query elastic.Query, typ interface{}, from int, size int,
	sorters ...elastic.Sorter) (data []interface{}, err error) {

	span, ctx := p.ZipkinNewSpan(ctx, "delete")
	if span != nil {
		defer span.Finish()
	}

	client, err := p.GetConn(ctx)
	if err != nil {
		return
	}

	res, err := client.Search().Index(p.Index).Type(p.Type).Query(query).
		From(from).Size(size).SortBy(sorters...).Do(ctx)
	if err != nil {
		log.Errorf("es search error :%s", err.Error())
	}

	data = res.Each(reflect.TypeOf(typ))

	return
}
