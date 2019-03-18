package zkha

import "log"

type Logger interface {
	DEBUGF(format string, v ...interface{})
	DEBUG(v ...interface{})
	INFOF(format string, v ...interface{})
	INFO(v ...interface{})
	WARNF(format string, v ...interface{})
	WARN(v ...interface{})
	ERRORF(format string, v ...interface{})
	ERROR(v ...interface{})
}

type defaultLogger struct{}

func (l *defaultLogger) DEBUGF(format string, v ...interface{}) { log.Printf(format, v...) }
func (l *defaultLogger) DEBUG(v ...interface{})                 { log.Print(v...) }
func (l *defaultLogger) INFOF(format string, v ...interface{})  { log.Printf(format, v...) }
func (l *defaultLogger) INFO(v ...interface{})                  { log.Print(v...) }
func (l *defaultLogger) WARNF(format string, v ...interface{})  { log.Printf(format, v...) }
func (l *defaultLogger) WARN(v ...interface{})                  { log.Print(v...) }
func (l *defaultLogger) ERRORF(format string, v ...interface{}) { log.Printf(format, v...) }
func (l *defaultLogger) ERROR(v ...interface{})                 { log.Print(v...) }

func init() {
	logger = &defaultLogger{}
}
