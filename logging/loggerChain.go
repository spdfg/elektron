package logging

import (
	log "github.com/sirupsen/logrus"
	"os"
)

type logInterface interface {
	setNext(logType logInterface)
	Log(logType int, level log.Level, message string)
	Logf(logType int, level log.Level, msgFmtString string, args ...interface{})
	createLogFile(prefix string)
}
type baseLogData struct {
	data log.Fields
}
type loggerImpl struct {
	*baseLogData
	logType        int
	allowOnConsole bool
	logFile        *os.File
	next           logInterface
	logger         *log.Logger
	logDir         *logDirectory
}

func (l *loggerImpl) WithFields(logData log.Fields) *loggerImpl {
	l.data = logData
	return l
}

func (l *loggerImpl) WithField(key string, value string) *loggerImpl {
	l.data[key] = value
	return l
}

func (l *loggerImpl) setNext(logType logInterface) {
	l.next = logType
}

func (l loggerImpl) Log(logType int, level log.Level, message string) {
	if l.next != nil {
		l.next.Log(logType, level, message)
	}
}

func (l loggerImpl) Logf(logType int, level log.Level, msgFmtString string, args ...interface{}) {
	if l.next != nil {
		l.next.Logf(logType, level, msgFmtString, args...)
	}
}

func (l *loggerImpl) resetFields() {
	l.data = nil
	l.data = log.Fields{}
}
