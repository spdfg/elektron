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

func NewSchedTraceLogger(b *baseLogData, logType int, prefix string) *SchedTraceLogger {
	sLog := &SchedTraceLogger{}
	sLog.Type = logType
	sLog.CreateLogFile(prefix)
	sLog.next = nil
	sLog.baseLogData = b
	return sLog
}

func (sLog SchedTraceLogger) Log(logType int, level log.Level, message string) {
	if sLog.Type == logType {
		if config.SchedTraceConfig.Enabled {
			if sLog.AllowOnConsole {
				logger.SetOutput(os.Stdout)
				logger.WithFields(sLog.data).Log(level, message)
			}

			logger.SetOutput(sLog.LogFile)
			logger.WithFields(sLog.data).Log(level, message)
		}
	}
	if sLog.next != nil {
		sLog.next.Log(logType, level, message)
	} else {
		// Clearing the fields.
		sLog.resetFields()
	}
}

func (sLog SchedTraceLogger) Logf(logType int, level log.Level, msgFmtString string, args ...interface{}) {
	if sLog.Type == logType {
		if config.SchedTraceConfig.Enabled {
			if sLog.AllowOnConsole {
				logger.SetOutput(os.Stdout)
				logger.WithFields(sLog.data).Logf(level, msgFmtString, args...)
			}

			logger.SetOutput(sLog.LogFile)
			logger.WithFields(sLog.data).Logf(level, msgFmtString, args...)
		}
	}
	// Forwarding to next logger
	if sLog.next != nil {
		sLog.next.Logf(logType, level, msgFmtString, args...)
	} else {
		// Clearing the fields.
		sLog.resetFields()
	}
}

func (sLog *SchedTraceLogger) CreateLogFile(prefix string) {
	if config.SchedTraceConfig.Enabled {
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
}
