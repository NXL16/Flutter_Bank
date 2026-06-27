package response

import (
	"github.com/gin-gonic/gin"
)

// Response định nghĩa cấu trúc JSON phản hồi chuẩn
type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// Success gửi phản hồi thành công (HTTP 20x hoặc 30x)
func Success(c *gin.Context, statusCode int, message string, data interface{}) {
	c.JSON(statusCode, Response{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// Error gửi phản hồi lỗi (HTTP 40x hoặc 50x)
func Error(c *gin.Context, statusCode int, message string, err error) {
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}
	c.JSON(statusCode, Response{
		Success: false,
		Message: message,
		Error:   errStr,
	})
}
