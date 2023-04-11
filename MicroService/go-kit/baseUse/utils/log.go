package utils

import (
	logTool "go-kit/logtol"
	"go.uber.org/zap"
)

var logger *zap.Logger // 全局

func NewLoggerServer() {
	logger = logTool.NewLogger(
		logTool.SetAppName("go-kit"),
		logTool.SetDevelopment(true),
		logTool.SetLevel(zap.DebugLevel),
	)
}

func GetLogger() *zap.Logger {
	return logger
}
