package elektronLogging

import (
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

type PCPLogger struct {
	loggerImpl
}

func NewPCPLogger(b *baseLogData, logType int, prefix string,
	logger *log.Logger, logDir *logDirectory) *PCPLogger {
	pLog := &PCPLogger{}
	pLog.logType = logType
	pLog.logDir = logDir
	pLog.next = nil
	pLog.baseLogData = b
	pLog.logger = logger
	pLog.createLogFile(prefix)
	return pLog
}

func (pLog PCPLogger) Log(logType int, level log.Level, message string) {
	if pLog.logType == logType {
		if config.PCPConfig.Enabled {
			if pLog.allowOnConsole {
				pLog.logger.SetOutput(os.Stdout)
				pLog.logger.WithFields(pLog.data).Log(level, message)
			}

			pLog.logger.SetOutput(pLog.logFile)
			pLog.logger.WithFields(pLog.data).Log(level, message)
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
	if pLog.logType == logType {
		if config.PCPConfig.Enabled {
			if pLog.allowOnConsole {
				pLog.logger.SetOutput(os.Stdout)
				pLog.logger.WithFields(pLog.data).Logf(level, msgFmtString, args...)
			}

			pLog.logger.SetOutput(pLog.logFile)
			pLog.logger.WithFields(pLog.data).Logf(level, msgFmtString, args...)
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

func (pLog *PCPLogger) createLogFile(prefix string) {
	if config.PCPConfig.Enabled {
		filename := strings.Join([]string{prefix, config.PCPConfig.FilenameExtension}, "")
		dirName := pLog.logDir.getDirName()
		if dirName != "" {
			if logFile, err := os.Create(filepath.Join(dirName, filename)); err != nil {
				log.Fatal("Unable to create logFile: ", err)
			} else {
				pLog.logFile = logFile
				pLog.allowOnConsole = config.PCPConfig.AllowOnConsole
			}
		}
	}
}
