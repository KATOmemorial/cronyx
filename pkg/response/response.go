package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Code int         `json:"code"` // 0:成功, 非0:错误
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

// Success 成功返回
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code: 0,
		Msg:  "success",
		Data: data,
	})
}

// Error 错误返回
func Error(c *gin.Context, httpCode int, msg string) {
	c.JSON(httpCode, Response{
		Code: httpCode, // 简单起见，业务错误码直接用 HTTP 状态码，也可以自定义
		Msg:  msg,
		Data: nil,
	})
}
