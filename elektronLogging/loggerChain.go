package elektronLogging

import (
	log "github.com/sirupsen/logrus"
	"os"
)

type Logger interface {
	SetNext(logType Logger)
	Log(logType int, level log.Level, message string)
	Logf(logType int, level log.Level, msgFmtString string, args ...interface{})
	CreateLogFile(prefix string)
}
type baseLogData struct {
	data log.Fields
}
type LoggerImpl struct {
	*baseLogData
	logType        int
	allowOnConsole bool
	logFile        *os.File
	next           Logger
	logger         *log.Logger
}

func (l *LoggerImpl) WithFields(logData log.Fields) *LoggerImpl {
	l.data = logData
	return l
}

func (l *LoggerImpl) WithField(key string, value string) *LoggerImpl {
	l.data[key] = value
	return l
}

func (l *LoggerImpl) SetNext(logType Logger) {
	l.next = logType
}

func (l LoggerImpl) Log(logType int, level log.Level, message string) {
	if l.next != nil {
		l.next.Log(logType, level, message)
	}
}

func (l LoggerImpl) Logf(logType int, level log.Level, msgFmtString string, args ...interface{}) {
	if l.next != nil {
		l.next.Logf(logType, level, msgFmtString, args...)
	}
}

func (l *LoggerImpl) resetFields() {
	l.data = nil
	l.data = log.Fields{}
}
