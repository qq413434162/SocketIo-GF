// Package core 框架用到的请求中间件
package core

import (
	"socketio-gf/utility/response"
	"net/http"

	"github.com/gogf/gf/v2/container/gset"

	"github.com/gogf/gf/v2/os/gtime"

	"github.com/gogf/gf/v2/errors/gerror"

	"github.com/gogf/gf/v2/errors/gcode"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

// Middleware 中间件管理服务
var Middleware = middlewareCore{}

// ShowDetailErrorCode 白名单内的错误码,能显示详细错误
var ShowDetailErrorCode = gset.NewIntSetFrom([]int{
	gcode.CodeValidationFailed.Code(),
})

type middlewareCore struct{}

// HandlerResponse 返回结果处理
func (s *middlewareCore) HandlerResponse(r *ghttp.Request) {
	begin := gtime.Now()
	g.Log().Infof(r.GetCtx(), "START: Run %v %v %.100v", r.RequestURI, r.Request.Method, r.GetRequestMap())
	r.Middleware.Next()
	g.Log().Infof(r.GetCtx(), "END: Run %v %v", r.RequestURI, gtime.Now().Sub(begin).String())
	// 如果已经有返回内容，那么该中间件什么也不做
	if r.Response.BufferLength() > 0 {
		return
	}

	err := r.GetError()
	if err != nil {
		// 当出错时显示更多的提交数据 最大上百m的字符
		g.Log().Warningf(r.GetCtx(), "REQUEST: Param %v %v %.1000000000", r.RequestURI, r.Request.Method, r.GetRequestMap())
		// 在白名单的错误码,直接显示错误明细
		if ShowDetailErrorCode.Contains(gerror.Code(err).Code()) {
			causeErr := gerror.Cause(err).Error()
			if causeErr != "" {
				response.JsonErrorForRequest(r, err, causeErr)
			}
			response.JsonErrorForRequest(r, err)
		} else {
			// 非生产模式才能暴露真正错误,不然很容易泄露内部信息,如sql相关
			if !Config.Env.Profile.Prod {
				causeErr := gerror.Cause(err).Error()
				if causeErr != "" {
					response.JsonErrorForRequest(r, err, causeErr)
				}
			}
			response.JsonErrorForRequest(r, err)
		}
	} else {
		var (
			res  = r.GetHandlerResponse()
			code = gerror.Code(err)
		)
		if r.Response.Status != http.StatusOK {
			switch r.Response.Status {
			case http.StatusNotFound:
				code = gcode.CodeNotFound
			case http.StatusForbidden:
				code = gcode.CodeNotAuthorized
			default:
				code = gcode.CodeUnknown
			}
			// 重新封装
			code = gcode.New(code.Code(), http.StatusText(r.Response.Status), nil)
		} else {
			code = gcode.CodeOK
		}
		response.JsonForRequestCode(r, code, res)
	}
}
