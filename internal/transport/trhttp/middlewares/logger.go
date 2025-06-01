package middlewares

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// LoggerMiddleware должен быть первый в стеке миддлваре.
func LoggerMiddleware(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if logger == nil {
			c.Next()
			return
		}

		start := time.Now()
		c.Next()
		latency := time.Since(start)

		statusCode := c.Writer.Status()
		l := logger.WithFields(logrus.Fields{
			"URI":              c.Request.RequestURI,
			"latency":          fmt.Sprintf("%d ms", latency.Milliseconds()),
			"status":           statusCode,
			"method":           c.Request.Method,
			"content-type":     c.Request.Header.Get("Content-Type"),
			"content-encoding": c.Request.Header.Get("Content-Encoding"),
		})
		errorMessage := c.Errors.ByType(gin.ErrorTypePrivate).String()

		if errorMessage != "" {
			l = l.WithField("error", errorMessage)
		}

		switch {
		case statusCode >= http.StatusInternalServerError:
			l.Error("Server error")
		case statusCode >= http.StatusBadRequest:
			l.Warn("Client error")
		default:
			l.Info("Request processed")
		}
	}
}
