package services

import (
	logger "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

type noopLogger struct{}

func (noopLogger) Debug(msg string, fields ...logger.Fields) {}
func (noopLogger) Info(msg string, fields ...logger.Fields)  {}
func (noopLogger) Warn(msg string, fields ...logger.Fields)  {}
func (noopLogger) Error(msg string, fields ...logger.Fields) {}
func (noopLogger) Fatal(msg string, fields ...logger.Fields) {}
func (noopLogger) WithFields(fields logger.Fields) logger.Logger {
	return noopLogger{}
}
