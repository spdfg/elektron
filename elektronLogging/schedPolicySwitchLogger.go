package elektronLogging

import (
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

type SchedPolicySwitchLogger struct {
	LoggerImpl
}

func NewSchedPolicySwitchLogger(logType int, prefix string) *SchedPolicySwitchLogger {
	sLog := &SchedPolicySwitchLogger{}
	sLog.Type = logType
	sLog.CreateLogFile(prefix)
	return sLog
}

func (sLog SchedPolicySwitchLogger) Log(logType int, level log.Level, logData log.Fields, message string) {
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

func (sLog *SchedPolicySwitchLogger) CreateLogFile(prefix string) {

	filename := strings.Join([]string{prefix, config.SPSConfig.FilenameExtension}, "")
	dirName := logDir.getDirName()
	if dirName != "" {
		if logFile, err := os.Create(filepath.Join(dirName, filename)); err != nil {
			log.Fatal("Unable to create logFile: ", err)
		} else {
			sLog.LogFile = logFile
			sLog.AllowOnConsole = config.SPSConfig.AllowOnConsole
		}
	}
}
