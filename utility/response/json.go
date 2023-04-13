package response

import (
	"strings"

	"github.com/gogf/gf/v2/errors/gerror"

	"github.com/gogf/gf/v2/errors/gcode"
	"github.com/gogf/gf/v2/net/ghttp"
)

// JsonResponse 数据返回通用JSON数据结构
type JsonResponse struct {
	Result  int         `json:"result"`  // 错误码((0:成功, 1:失败, >1:错误码))
	Message string      `json:"message"` // 提示信息
	Data    interface{} `json:"data"`    // 返回数据(业务接口定义具体数据结构)
}

// JsonForRequestCode 标准返回结果数据结构封装。
func JsonForRequestCode(r *ghttp.Request, code gcode.Code, data ...interface{}) {
	responseData := interface{}(nil)
	if len(data) > 0 {
		responseData = data[0]
	}
	r.Response.WriteJson(JsonResponse{
		Result:  code.Code(),
		Message: code.Message(),
		Data:    responseData,
	})
}

func JsonErrorForRequest(r *ghttp.Request, err error, message ...string) {
	code := gerror.Code(err)
	useMessage := code.Message()
	if len(message) > 0 {
		useMessage += ": " + strings.Join(message, ";")
	}
	c := gcode.New(code.Code(), useMessage, nil)
	JsonForRequestCode(r, c)
	r.Exit()
}