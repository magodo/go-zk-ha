package zkha

import "log"

type Logger interface {
	Debugf(format string, v ...interface{})
	Debug(v ...interface{})
	Infof(format string, v ...interface{})
	Info(v ...interface{})
	Warnf(format string, v ...interface{})
	Warn(v ...interface{})
	Errorf(format string, v ...interface{})
	Error(v ...interface{})
}

type defaultLogger struct{}

func (l *defaultLogger) Debugf(format string, v ...interface{}) { log.Printf(format, v...) }
func (l *defaultLogger) Debug(v ...interface{})                 { log.Print(v...) }
func (l *defaultLogger) Infof(format string, v ...interface{})  { log.Printf(format, v...) }
func (l *defaultLogger) Info(v ...interface{})                  { log.Print(v...) }
func (l *defaultLogger) Warnf(format string, v ...interface{})  { log.Printf(format, v...) }
func (l *defaultLogger) Warn(v ...interface{})                  { log.Print(v...) }
func (l *defaultLogger) Errorf(format string, v ...interface{}) { log.Printf(format, v...) }
func (l *defaultLogger) Error(v ...interface{})                 { log.Print(v...) }

func init() {
	logger = &defaultLogger{}
}
