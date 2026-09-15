package handlers

import (
	"log"

	"github.com/gin-gonic/gin"
)

const (
	ServerMsg  = "Interserver failed!"
	ParaMsg    = "Parameter reception failed!"
	ParaPidMsg = "Parameter reception failed or pid is not uint"
	FindMsg    = "Product not found!"
)

// 在序列化 Go 结构体时，如果字段值为“空”Go 值，则标记为 `omitempty` 的字段会被省略。
// 这些“空”值包括：false、0、nil 指针、nil 接口值，以及任何空数组、切片、映射或字符串。
type Response struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data,omitempty"` // omitempty 表示没数据时不展示这个字段
}

func Success(c *gin.Context, msg string, data any) {
	c.JSON(200, Response{
		Code: 200,
		Msg:  msg,
		Data: data,
	})
}

func FailResponse(c *gin.Context, code int, msg string, err error) {
	// 5xx内部错误细节只进日志,不传给客户端(防信息泄露); 4xx业务错可带原因
	var detail any
	if err != nil && code < 500 {
		detail = err.Error()
	}
	if err != nil && code >= 500 {
		log.Printf("500 on %s: %v", c.FullPath(), err)
	}
	c.JSON(code, Response{
		Code: code,
		Msg:  msg,
		Data: detail,
	})
}
