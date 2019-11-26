package elektronLogging

import (
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

type SchedTraceLogger struct {
	LoggerImpl
}

func NewSchedTraceLogger(logType int, prefix string) *SchedTraceLogger {
	sLog := &SchedTraceLogger{}
	sLog.Type = logType
	sLog.CreateLogFile(prefix)
	return sLog
}

func (sLog SchedTraceLogger) Log(logType int, level log.Level, logData log.Fields, message string) {
	if sLog.Type == logType {

		logger.SetLevel(level)

		if sLog.AllowOnConsole {
			logger.SetOutput(os.Stdout)
			logger.WithFields(logData).Println(message)
		}

		logger.SetOutput(sLog.LogFile)
		logger.WithFields(logData).Println(message)
	}
	if sLog.next != nil {
		sLog.next.Log(logType, level, logData, message)
	}
}

func (sLog *SchedTraceLogger) CreateLogFile(prefix string) {

	filename := strings.Join([]string{prefix, config.SchedTraceConfig.FilenameExtension}, "")
	dirName := logDir.getDirName()
	if dirName != "" {
		if logFile, err := os.Create(filepath.Join(dirName, filename)); err != nil {
			log.Fatal("Unable to create logFile: ", err)
		} else {
			sLog.LogFile = logFile
			sLog.AllowOnConsole = config.SchedTraceConfig.AllowOnConsole
		}
	}
}
