package dao

import (
	"net/url"
	"context"
	"net/http"
	"time"
	"fmt"
	"github.com/tonyjt/tgo_v2/config"
	"github.com/tonyjt/tgo_v2/log"
	"github.com/tonyjt/tgo_v2/terror"
	"github.com/tonyjt/tgo_v2/pconst"
	"github.com/opentracing/opentracing-go"
	"strings"
	"gopkg.in/mgo.v2"
)

type Http struct {
	Service string
	Url string
	Timeout time.Duration
}

func init(){
	if config.FeatureMongo() {

		for _,c:= range config.MongoGetAll(){
			configMongo := c.Conn
			if strings.Trim(configMongo.ReadOption, " ") == "" {
				configMongo.ReadOption = "nearest"
			}

			connectionString := fmt.Sprintf("mongodb://%s", configMongo.Servers)

			var err error
			sessionMongo, err = mgo.Dial(connectionString)

			if err != nil {

				log.Logf(log.LevelFatal, "connect to mongo server error:%s,%s", err.Error(), connectionString)
				panic("connect to mongo server")

			}
			sessionMongo.SetPoolLimit(configMongo.PoolLimit)
		}

	}
}

func (p *Http) ZipkinNewSpan(ctx context.Context,name string)(opentracing.Span,context.Context){
	if config.FeatureZipkin(){
		return opentracing.StartSpanFromContext(ctx,fmt.Sprintf("http:%s",name))
	}else{
		return nil,ctx
	}
}
func (p *Http) proccessError(span opentracing.Span,err error,msg string)(error){
	log.Error(msg)
	if span !=nil{
		span.SetTag("err",err)
	}
	return err
}
// PostForm
func (p *Http) PostForm(ctx context.Context,path string,data url.Values)(response *http.Response,err error){

	span,ctx := p.ZipkinNewSpan(ctx,path)

	if span!=nil{
		defer span.Finish()
	}

	u := p.Url + path


	client := http.Client{Timeout:p.Timeout}

	response,err = client.PostForm(u,data)

	if err!=nil{
		msg:= fmt.Sprintf("post form url:%s,err:%s",u,err.Error())
		err = terror.New(pconst.ERROR_HTTP_POSTFORM)
		p.proccessError(span,err,msg)
	}else if response == nil {
		msg:= fmt.Sprintf("post form url:%s,response is nil",u)
		err = terror.New(pconst.ERROR_HTTP_POSTFORM_RESPONSE)
		p.proccessError(span,err,msg)
	}

	return
}

// Get
func (p *Http) Get(ctx context.Context,path string)(response *http.Response,err error){
	span,ctx := p.ZipkinNewSpan(ctx,path)

	if span!=nil{
		defer span.Finish()
	}

	u := p.Url + path


	client := http.Client{Timeout:p.Timeout}


	response,err = client.Get(u)

	if err!=nil{
		msg:= fmt.Sprintf("get url:%s,err:%s",u,err.Error())
		err = terror.New(pconst.ERROR_HTTP_POSTFORM)
		p.proccessError(span,err,msg)
	}else if response == nil {
		msg:= fmt.Sprintf("post form url:%s,response is nil",u)
		err = terror.New(pconst.ERROR_HTTP_POSTFORM_RESPONSE)
		p.proccessError(span,err,msg)
	}
	return
}

