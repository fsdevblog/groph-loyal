package logger

import (
	"io"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

// New инициализирует логгер.
func New(output io.Writer) *logrus.Logger {
	l := logrus.New()
	l.SetOutput(output)
	l.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339Nano,
	})
	l.SetLevel(logrus.InfoLevel)

	// перезаписываем ряд настроек для окружений отличных от продакшн
	if os.Getenv("GIN_MODE") != "release" {
		l.SetLevel(logrus.DebugLevel)
		l.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
	}

	return l
}
