package elektronLogging

import (
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

type SchedWindowLogger struct {
	LoggerImpl
}

func NewSchedWindowLogger(b *baseLogData, logType int, prefix string) *SchedWindowLogger {
	sLog := &SchedWindowLogger{}
	sLog.Type = logType
	sLog.CreateLogFile(prefix)
	sLog.next = nil
	sLog.baseLogData = b
	return sLog
}

func (sLog SchedWindowLogger) Log(logType int, level log.Level, message string) {
	if config.SchedWindowConfig.Enabled {
		if sLog.Type == logType {

			if sLog.AllowOnConsole {
				logger.SetOutput(os.Stdout)
				logger.WithFields(sLog.data).Log(level, message)
			}

			logger.SetOutput(sLog.LogFile)
			logger.WithFields(sLog.data).Log(level, message)
		}
		if sLog.next != nil {
			sLog.next.Log(logType, level, message)
		} else {
			// Clearing the fields.
			sLog.resetFields()
		}
	}
}

func (sLog SchedWindowLogger) Logf(logType int, level log.Level, msgFmtString string, args ...interface{}) {
	if config.SchedWindowConfig.Enabled {
		if sLog.Type == logType {

			if sLog.AllowOnConsole {
				logger.SetOutput(os.Stdout)
				logger.WithFields(sLog.data).Logf(level, msgFmtString, args...)
			}

			logger.SetOutput(sLog.LogFile)
			logger.WithFields(sLog.data).Logf(level, msgFmtString, args...)
		}
		if sLog.next != nil {
			sLog.next.Logf(logType, level, msgFmtString, args...)
		} else {
			// Clearing the fields.
			sLog.resetFields()
		}
	}
}

func (sLog *SchedWindowLogger) CreateLogFile(prefix string) {
	if config.SchedWindowConfig.Enabled {
		filename := strings.Join([]string{prefix, config.SchedWindowConfig.FilenameExtension}, "")
		dirName := logDir.getDirName()
		if dirName != "" {
			if logFile, err := os.Create(filepath.Join(dirName, filename)); err != nil {
				log.Fatal("Unable to create logFile: ", err)
			} else {
				sLog.LogFile = logFile
				sLog.AllowOnConsole = config.SchedWindowConfig.AllowOnConsole
			}
		}
	}
}
