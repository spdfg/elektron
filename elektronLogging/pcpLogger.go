package elektronLogging

import (
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

type PCPLogger struct {
	LoggerImpl
}

func NewPCPLogger(b *baseLogData, logType int, prefix string) *PCPLogger {
	pLog := &PCPLogger{}
	pLog.Type = logType
	pLog.CreateLogFile(prefix)
	pLog.next = nil
	pLog.baseLogData = b
	return pLog
}

func (pLog PCPLogger) Log(logType int, level log.Level, message string) {
	if pLog.Type == logType {
		if config.PCPConfig.Enabled {
			if pLog.AllowOnConsole {
				logger.SetOutput(os.Stdout)
				logger.WithFields(pLog.data).Log(level, message)
			}

			logger.SetOutput(pLog.LogFile)
			logger.WithFields(pLog.data).Log(level, message)
		}
	}
	if pLog.next != nil {
		pLog.next.Log(logType, level, message)
	} else {
		// Clearing the fields.
		pLog.resetFields()
	}
}

func (pLog PCPLogger) Logf(logType int, level log.Level, msgFmtString string, args ...interface{}) {
	if pLog.Type == logType {
		if config.PCPConfig.Enabled {
			if pLog.AllowOnConsole {
				logger.SetOutput(os.Stdout)
				logger.WithFields(pLog.data).Logf(level, msgFmtString, args...)
			}

			logger.SetOutput(pLog.LogFile)
			logger.WithFields(pLog.data).Logf(level, msgFmtString, args...)
		}
	}
	// Forwarding to next logger
	if pLog.next != nil {
		pLog.next.Logf(logType, level, msgFmtString, args...)
	} else {
		// Clearing the fields.
		pLog.resetFields()
	}
}

func (pLog *PCPLogger) CreateLogFile(prefix string) {
	if config.PCPConfig.Enabled {
		filename := strings.Join([]string{prefix, config.PCPConfig.FilenameExtension}, "")
		dirName := logDir.getDirName()
		if dirName != "" {
			if logFile, err := os.Create(filepath.Join(dirName, filename)); err != nil {
				log.Fatal("Unable to create logFile: ", err)
			} else {
				pLog.LogFile = logFile
				pLog.AllowOnConsole = config.PCPConfig.AllowOnConsole
			}
		}
	}
}
