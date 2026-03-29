package resp

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Body struct {
	Code      int         `json:"code"`
	Message   string      `json:"message"`
	ErrorCode string      `json:"errorCode,omitempty"`
	Data      interface{} `json:"data,omitempty"`
}

func OK(c *gin.Context, data interface{}) {
	if data == nil {
		data = map[string]interface{}{}
	}
	c.JSON(http.StatusOK, Body{Code: 0, Message: "success", Data: data})
}

func Err(c *gin.Context, httpStatus int, bizCode int, message, errorCode string, data interface{}) {
	if data == nil {
		data = map[string]interface{}{}
	}
	c.JSON(httpStatus, Body{
		Code:      bizCode,
		Message:   message,
		ErrorCode: errorCode,
		Data:      data,
	})
}

func Unauthorized(c *gin.Context, msg string) {
	Err(c, http.StatusUnauthorized, 40100, msg, "E_UNAUTHORIZED", nil)
}

func BadRequest(c *gin.Context, msg, code string, data map[string]interface{}) {
	Err(c, http.StatusBadRequest, 10001, msg, code, data)
}

func NotFound(c *gin.Context, msg string) {
	Err(c, http.StatusNotFound, 10004, msg, "E_NOT_FOUND", nil)
}

func Conflict(c *gin.Context, msg, code string) {
	Err(c, http.StatusConflict, 10009, msg, code, nil)
}

func TooManyRequests(c *gin.Context, msg, code string) {
	Err(c, http.StatusTooManyRequests, 10029, msg, code, nil)
}

func Internal(c *gin.Context, msg string) {
	Err(c, http.StatusInternalServerError, 10500, msg, "E_NETWORK", nil)
}
