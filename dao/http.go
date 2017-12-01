package dao

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/opentracing/opentracing-go"
	"github.com/tonyjt/tgo_v2/config"
	"github.com/tonyjt/tgo_v2/log"
	"github.com/tonyjt/tgo_v2/pconst"
	"github.com/tonyjt/tgo_v2/terror"
	"io/ioutil"
	"net/http"
	"net/url"
)

type Http struct {
	Service string
}

func (p *Http) ZipkinNewSpan(ctx context.Context, name string) (opentracing.Span, context.Context) {
	if config.FeatureZipkin() {
		return opentracing.StartSpanFromContext(ctx, fmt.Sprintf("http:%s/%s", p.Service, name))
	} else {
		return nil, ctx
	}
}
func (p *Http) proccessError(span opentracing.Span, err error, msg string) error {
	log.Error(msg)
	if span != nil {
		span.SetTag("err", err)
	}
	return err
}

// PostForm
func (p *Http) PostForm(ctx context.Context, pathKey string, data url.Values) (response *http.Response, err error) {

	span, ctx := p.ZipkinNewSpan(ctx, pathKey)

	if span != nil {
		defer span.Finish()
	}

	conf := config.HttpGet(p.Service)

	u, err := p.url(ctx, span, conf, pathKey)

	if err != nil {
		return
	}

	client := http.Client{Timeout: conf.Conn.Timeout}

	response, err = client.PostForm(u, data)

	if err != nil {
		msg := fmt.Sprintf("post form url:%s,err:%s", u, err.Error())
		err = terror.New(pconst.ERROR_HTTP_POSTFORM)
		p.proccessError(span, err, msg)
	} else if response == nil {
		msg := fmt.Sprintf("post form url:%s,response is nil", u)
		err = terror.New(pconst.ERROR_HTTP_POSTFORM_RESPONSE)
		p.proccessError(span, err, msg)
	}

	return
}

// PostFormAndReadAll
func (p *Http) PostFormAndReadAll(ctx context.Context, pathKey string, data url.Values) (body []byte, err error) {

	resp, err := p.PostForm(ctx, pathKey, data)

	if err != nil {
		return
	}

	body, err = ioutil.ReadAll(resp.Body)

	if err != nil {
		log.Errorf("post form read error:%s", err.Error())

		err = terror.New(pconst.ERROR_HTTP_READ)
	}

	return
}

// Get
func (p *Http) Get(ctx context.Context, pathKey string, queryString string) (response *http.Response, err error) {
	span, ctx := p.ZipkinNewSpan(ctx, pathKey)

	if span != nil {
		defer span.Finish()
	}

	conf := config.HttpGet(p.Service)

	u, err := p.url(ctx, span, conf, pathKey)

	if err != nil {
		return
	}

	client := http.Client{Timeout: conf.Conn.Timeout}

	response, err = client.Get(fmt.Sprintf("%s?%s", u, queryString))

	if err != nil {
		msg := fmt.Sprintf("get url:%s,err:%s", u, err.Error())
		err = terror.New(pconst.ERROR_HTTP_POSTFORM)
		p.proccessError(span, err, msg)
	} else if response == nil {
		msg := fmt.Sprintf("post form url:%s,response is nil", u)
		err = terror.New(pconst.ERROR_HTTP_POSTFORM_RESPONSE)
		p.proccessError(span, err, msg)
	}
	return
}

// GetAndReadAll
func (p *Http) GetAndReadAll(ctx context.Context, pathKey string, queryString string) (body []byte, err error) {

	resp, err := p.Get(ctx, pathKey, queryString)

	if err != nil {
		return
	}

	body, err = ioutil.ReadAll(resp.Body)

	if err != nil {
		log.Errorf("get read error:%s", err.Error())

		err = terror.New(pconst.ERROR_HTTP_READ)
	}

	return
}

// UnmarshalBody
func (p *Http) UnmarshalBody(ctx context.Context, body []byte, data interface{}) (err error) {
	err = json.Unmarshal(body, data)

	if err != nil {
		log.Errorf("json unmarshal failed,error:%s", err.Error())
		err = terror.New(pconst.ERROR_HTTP_UNMARSHAL)
	}
	return
}

func (p *Http) url(ctx context.Context, span opentracing.Span, conf *config.HttpConf, pathKey string) (url string, err error) {

	if conf == nil {
		msg := fmt.Sprintf("post form %s,config is nil", p.Service)
		err = terror.New(pconst.ERROR_HTTP_CONFIG)
		p.proccessError(span, err, msg)
		return
	}

	path := config.HttpGetPath(conf, pathKey)

	if path == "" {
		msg := fmt.Sprintf("post form %s,path is nil", pathKey)
		err = terror.New(pconst.ERROR_HTTP_CONFIG)
		p.proccessError(span, err, msg)
	}

	url = conf.Conn.Url + path

	return
}
