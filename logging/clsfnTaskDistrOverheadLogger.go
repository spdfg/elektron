package logging

import (
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

type ClsfnTaskDistrOverheadLogger struct {
	loggerImpl
}

func NewClsfnTaskDistrOverheadLogger(b *baseLogData, logType int, prefix string,
	logger *log.Logger, logDir *logDirectory) *ClsfnTaskDistrOverheadLogger {
	cLog := &ClsfnTaskDistrOverheadLogger{}
	cLog.logType = logType
	cLog.logDir = logDir
	cLog.next = nil
	cLog.baseLogData = b
	cLog.logger = logger
	cLog.createLogFile(prefix)
	return cLog
}

func (cLog ClsfnTaskDistrOverheadLogger) Log(logType int, level log.Level, message string) {
	if cLog.logType == logType {
		if config.TaskDistrConfig.Enabled {
			if cLog.allowOnConsole {
				cLog.logger.SetOutput(os.Stdout)
				cLog.logger.WithFields(cLog.data).Log(level, message)
			}

			cLog.logger.SetOutput(cLog.logFile)
			cLog.logger.WithFields(cLog.data).Log(level, message)
		}
	}
	// Forwarding to next logger
	if cLog.next != nil {
		cLog.next.Log(logType, level, message)
	} else {
		// Clearing the fields.
		cLog.resetFields()
	}
}

func (cLog ClsfnTaskDistrOverheadLogger) Logf(logType int, level log.Level, msgFmtString string, args ...interface{}) {
	if cLog.logType == logType {
		if config.TaskDistrConfig.Enabled {
			if cLog.allowOnConsole {
				cLog.logger.SetOutput(os.Stdout)
				cLog.logger.WithFields(cLog.data).Logf(level, msgFmtString, args...)
			}

			cLog.logger.SetOutput(cLog.logFile)
			cLog.logger.WithFields(cLog.data).Logf(level, msgFmtString, args...)
		}
	}
	if cLog.next != nil {
		cLog.next.Logf(logType, level, msgFmtString, args...)
	} else {
		// Clearing the fields.
		cLog.resetFields()
	}
}

func (cLog *ClsfnTaskDistrOverheadLogger) createLogFile(prefix string) {

	if config.TaskDistrConfig.Enabled {
		filename := strings.Join([]string{prefix, config.TaskDistrConfig.FilenameExtension}, "")
		dirName := cLog.logDir.getDirName()
		if dirName != "" {
			if logFile, err := os.Create(filepath.Join(dirName, filename)); err != nil {
				log.Fatal("Unable to create logFile: ", err)
			} else {
				cLog.logFile = logFile
				cLog.allowOnConsole = config.TaskDistrConfig.AllowOnConsole
			}
		}
	}
}
