package elektronLogging

import (
	log "github.com/sirupsen/logrus"
	"os"
	"strings"
)

type SchedTraceLogger struct {
	LoggerImpl
}

func NewSchedTraceLogger(logType int, prefix string) *SchedTraceLogger {
	sLog := &SchedTraceLogger{}
	sLog.Type = logType
	sLog.SetLogFile(prefix)
	return sLog
}

func (sLog *SchedTraceLogger) Log(logType int, level log.Level, logData log.Fields, message string) {
	if sLog.Type == logType {

		logger.SetLevel(level)

		if sLog.AllowOnConsole {
			logger.SetOutput(os.Stdout)
			logger.WithFields(logData).Println(message)
		}

		logger.SetOutput(sLog.LogFileName)
		logger.WithFields(logData).Println(message)
	}
	if sLog.next != nil {
		sLog.next.Log(logType, level, logData, message)
	}
}

func (sLog *SchedTraceLogger) SetLogFile(prefix string) {

	schedTraceLogPrefix := strings.Join([]string{prefix, config.SchedTraceConfig.FilenameExtension}, "")
	dirName := logDir.getDirName()
	if dirName != "" {
		schedTraceLogPrefix = strings.Join([]string{dirName, schedTraceLogPrefix}, "/")
	}
	if logFile, err := os.Create(schedTraceLogPrefix); err != nil {
		log.Fatal("Unable to create logFile: ", err)
	} else {
		sLog.LogFileName = logFile
		sLog.AllowOnConsole = config.SchedTraceConfig.AllowOnConsole
	}
}
