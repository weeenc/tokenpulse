// Package api 定义服务端统一的 HTTP 响应格式和输出方法。
package api

import "github.com/gin-gonic/gin"

// Response 是所有 JSON 接口共用的响应信封。
type Response struct {
	Code    int    `json:"code"`    // Code 为业务状态码，0 表示成功。
	Message string `json:"message"` // Message 为面向调用方的结果说明。
	Data    any    `json:"data"`    // Data 承载成功结果，失败时通常为 nil。
}

// OK 输出业务成功响应，并使用调用方指定的 HTTP 状态码。
func OK(c *gin.Context, status int, data any) {
	c.JSON(status, Response{Code: 0, Message: "success", Data: data})
}

// Error 中止当前 Gin 处理链并输出标准错误响应。
func Error(c *gin.Context, status, code int, message string) {
	c.AbortWithStatusJSON(status, Response{Code: code, Message: message, Data: nil})
}
