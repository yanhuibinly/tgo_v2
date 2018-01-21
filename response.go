package tgo_v2

import (
	"bytes"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/tonyjt/tgo_v2/config"
	"github.com/tonyjt/tgo_v2/log"
	"github.com/tonyjt/tgo_v2/pconst"
	"github.com/tonyjt/tgo_v2/terror"
	"net/http"
	"strings"
)

//ResponseReturnJSONNoP return json response without jsonp
func ResponseReturnJSONNoP(c *gin.Context, err error, model interface{}) {

	ResponseJSONWithCallbackFlag(c, err, model, false)
}

//ResponseJson json response
func ResponseJson(c *gin.Context, err error, model interface{}) {
	ResponseJSONWithCallbackFlag(c, err, model, true)
}

//ResponseJSONWithCallbackFlag  json response base func
func ResponseJSONWithCallbackFlag(c *gin.Context, err error, model interface{}, callbackFlag bool) {
	var rj interface{}

	var te *terror.TError
	var ok bool
	if err == nil {
		te = terror.New(pconst.ERROR_OK)
	} else {
		if te, ok = err.(*terror.TError); !ok {
			te = terror.NewFromError(err)
		}
		if te.Code == 0 {
			te.Code = 1001
		}
	}

	//添加结果
	if te.Level == terror.LevelException {
		c.Set("result", false)
	} else {
		c.Set("result", true)
	}

	if strings.Trim(te.Msg, " ") == "" {
		te.Msg = config.CodeGetMsg(te.Code)
	}

	configResp := config.RespGet()

	rj = gin.H{
		configResp.Code: te.Code,
		configResp.Msg:  te.GetMsg(),
		configResp.Data: model,
	}

	var callback string
	if callbackFlag {
		callback = c.Query("callback")
	}

	if strings.Trim(callback, " ") == "" {
		c.Status(200)

		header := c.Writer.Header()
		if val := header["Content-Type"]; len(val) == 0 {
			header["Content-Type"] = []string{"application/json; charset=utf-8"}
		}

		encoder := json.NewEncoder(c.Writer)
		encoder.SetEscapeHTML(false)
		err := encoder.Encode(rj)
		if err != nil {
			panic(err)
		}
	} else {
		b, err := responseJSONMarshal(rj)
		if err != nil {
			log.Errorf("jsonp marshal error:%s", err.Error())
		} else {
			c.String(200, "%s(%s)", callback, string(b))
		}
	}
}

//responseJSONMarshal response json marshal
func responseJSONMarshal(t interface{}) ([]byte, error) {
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(t)
	return buffer.Bytes(), err
}

//ResponseRedirect redirect
func ResponseRedirect(c *gin.Context, url string) {
	c.Redirect(http.StatusMovedPermanently, url)
}

//ResponseGrpc response grpc
func ResponseGrpc(err error) (code int64, msg string) {

	var codeint int
	if err == nil {
		codeint = pconst.ERROR_OK
	} else {
		var te *terror.TError
		var ok bool
		if te, ok = err.(*terror.TError); !ok {
			te = terror.NewFromError(err)
		}
		if te.Code == 0 {
			te.Code = pconst.ERROR_OK
		}
		codeint = te.Code
	}

	msg = config.CodeGetMsg(codeint)
	code = int64(codeint)
	return
}
