package elektronLogging

import (
	elekLog "github.com/sirupsen/logrus"
	"os"
	"strings"
)

type SchedWindowLogger struct {
	LoggerImpl
}

func NewSchedWindowLogger(logType int, prefix string) *SchedWindowLogger {
	sLog := &SchedWindowLogger{}
	sLog.Type = logType
	sLog.SetLogFile(prefix)
	return sLog
}

func (sLog *SchedWindowLogger) Log(logType int, level elekLog.Level, logData elekLog.Fields, message string) {
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

func (sLog *SchedWindowLogger) SetLogFile(prefix string) {

	schedWindowLogPrefix := strings.Join([]string{prefix, config.SchedWindowConfig.FilenameExtension}, "")
	dirName := logDir.getDirName()
	if dirName != "" {
		schedWindowLogPrefix = strings.Join([]string{dirName, schedWindowLogPrefix}, "/")
	}
	if logFile, err := os.Create(schedWindowLogPrefix); err != nil {
		elekLog.Fatal("Unable to create logFile: ", err)
	} else {
		sLog.LogFileName = logFile
		sLog.AllowOnConsole = config.SchedWindowConfig.AllowOnConsole
	}
}
