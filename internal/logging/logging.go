// Package logging настраивает структурированный логгер приложения на базе zap.
package logging

import (
	"go.uber.org/zap"
)

var logger *zap.Logger
var Sugar *zap.SugaredLogger

func LoggingInit() {

	var err error
	logger, err = zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	Sugar = logger.Sugar()
}

func LoggingDone() {
	logger.Sync()
}
