package elektronLogging

import (
	"os"
	//data "github.com/spdfg/elektron/elektronLogging/data"
	log "github.com/sirupsen/logrus"
)

type Logger interface {
	SetNext(logType Logger)
	Log(logType int, level log.Level, logData log.Fields, message string)
	SetLogFile(prefix string)
}
type LoggerImpl struct {
	Type        int
    AllowOnConsole bool
	LogFileName *os.File
	next        Logger
}

func (l *LoggerImpl) SetNext(logType Logger) {
	l.next = logType
}

func (l *LoggerImpl) Log(logType int, level log.Level, logData log.Fields, message string) {
	if l.next != nil {
		l.next.Log(logType, level, logData, message)
	}
}

/*func cloneFields(logData data.LogData) log.Fields {
	var newMap  = make(log.Fields)
	for k,v := range logData {
		newMap[k] = v
	}
	return newMap
}*/
