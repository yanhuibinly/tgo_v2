package zipkin

import (
	"github.com/gin-gonic/gin"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

func MiddlewardHttp() gin.HandlerFunc {

	return func(c *gin.Context) {

		tracer := opentracing.GlobalTracer()

		wireContext, err := tracer.Extract(
			opentracing.TextMap,
			opentracing.HTTPHeadersCarrier(c.Request.Header),
		)

		var span opentracing.Span
		on := c.Request.URL.Path
		if err != nil {
			span = opentracing.StartSpan(on)
		} else {
			span = tracer.StartSpan(on, ext.RPCServerOption(wireContext))
		}

		span.SetTag("server-http", "here")

		defer span.Finish()
		ctx := opentracing.ContextWithSpan(c.Request.Context(), span)

		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}
