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
		c.Next() // Сначала обработаем все остальные middleware и хендлеры. Соберем ошибки..

		if len(c.Errors) == 0 {
			return
		}

		// Обрабатываем только первую ошибку. В будущем, додумаю как отображать все ошибки.
		firstErr := c.Errors[0]
		var msg string
		// публичную ошибку отображаем
		if firstErr.IsType(gin.ErrorTypePublic) {
			msg = firstErr.Error()
		} else {
			// Для любого другого типа ошибки - отображаем заглушку.
			// Планируется доработать, чтоб внятно отображать bind ошибки валидатора, а пока просто оставлю заглушку.
			msg = statusErrorText(c.Writer.Status())
		}

		// отображаем ошибку в json или текстовом виде, в зависимости от заголовков запроса.
		accept := c.GetHeader("Accept")
		contentType := c.GetHeader("Content-Type")
		switch {
		case strings.Contains(accept, "application/json"),
			strings.Contains(contentType, "application/json"):
			c.JSON(c.Writer.Status(), gin.H{"error": msg})
		default:
			c.String(c.Writer.Status(), msg)
		}
		c.Abort()
	}
}
