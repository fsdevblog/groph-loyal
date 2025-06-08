package middlewares

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func statusErrorText(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "bad request"
	case http.StatusUnauthorized:
		return "unauthorized"
	case http.StatusForbidden:
		return "forbidden"
	case http.StatusNotFound:
		return "not found"
	case http.StatusUnprocessableEntity:
		return "unprocessable entity"
	case http.StatusConflict:
		return "conflict"
	default:
		return "internal server error"
	}
}

func Errors() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 {
			return
		}

		// обрабатываем только первую ошибку
		firstErr := c.Errors[0]
		var msg string
		if firstErr.IsType(gin.ErrorTypePublic) {
			msg = firstErr.Error()
		} else {
			msg = statusErrorText(c.Writer.Status())
		}

		accept := c.GetHeader("Accept")
		contentType := c.GetHeader("Content-Type")
		switch {
		case strings.Contains(accept, "application/json"),
			strings.Contains(contentType, "application/json"):
			c.JSON(c.Writer.Status(), gin.H{"error": msg})
		case strings.Contains(accept, "text/plain"):
			c.String(c.Writer.Status(), msg)
		default:
			c.String(c.Writer.Status(), msg)
		}
		c.Abort()
	}
}
