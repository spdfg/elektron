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

func NewSchedTraceLogger(b *baseLogData, logType int, prefix string, logger *log.Logger) *SchedTraceLogger {
	sLog := &SchedTraceLogger{}
	sLog.logType = logType
	sLog.CreateLogFile(prefix)
	sLog.next = nil
	sLog.baseLogData = b
	sLog.logger = logger
	return sLog
}

func (sLog SchedTraceLogger) Log(logType int, level log.Level, message string) {
	if sLog.logType == logType {
		if config.SchedTraceConfig.Enabled {
			if sLog.allowOnConsole {
				sLog.logger.SetOutput(os.Stdout)
				sLog.logger.WithFields(sLog.data).Log(level, message)
			}

			sLog.logger.SetOutput(sLog.logFile)
			sLog.logger.WithFields(sLog.data).Log(level, message)
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
	if sLog.logType == logType {
		if config.SchedTraceConfig.Enabled {
			if sLog.allowOnConsole {
				sLog.logger.SetOutput(os.Stdout)
				sLog.logger.WithFields(sLog.data).Logf(level, msgFmtString, args...)
			}

			sLog.logger.SetOutput(sLog.logFile)
			sLog.logger.WithFields(sLog.data).Logf(level, msgFmtString, args...)
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
				sLog.logFile = logFile
				sLog.allowOnConsole = config.SchedTraceConfig.AllowOnConsole
			}
		}
	}
}
