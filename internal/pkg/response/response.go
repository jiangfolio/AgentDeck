package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Body 统一 API 响应结构。
type Body struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// OK 返回成功响应。
func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Body{Code: 0, Message: "ok", Data: data})
}

// Fail 返回业务失败（HTTP 仍 200，靠 code 区分）。
func Fail(c *gin.Context, code int, message string) {
	if code == 0 {
		code = 1
	}
	c.JSON(http.StatusOK, Body{Code: code, Message: message, Data: nil})
}

// AbortFail 中断后续中间件并返回失败。
func AbortFail(c *gin.Context, httpStatus int, code int, message string) {
	c.AbortWithStatusJSON(httpStatus, Body{Code: code, Message: message, Data: nil})
}
