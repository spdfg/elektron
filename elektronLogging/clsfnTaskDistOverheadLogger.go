package elektronLogging

import (
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

type ClsfnTaskDistrOverheadLogger struct {
	LoggerImpl
}

func NewClsfnTaskDistrOverheadLogger(b *baseLogData, logType int, prefix string) *ClsfnTaskDistrOverheadLogger {
	cLog := &ClsfnTaskDistrOverheadLogger{}
	cLog.Type = logType
	cLog.CreateLogFile(prefix)
	cLog.next = nil
	cLog.baseLogData = b
	return cLog
}

func (cLog ClsfnTaskDistrOverheadLogger) Log(logType int, level log.Level, message string) {
	if cLog.Type == logType {
		if config.TaskDistrConfig.Enabled {
			if cLog.AllowOnConsole {
				logger.SetOutput(os.Stdout)
				logger.WithFields(cLog.data).Log(level, message)
			}

			logger.SetOutput(cLog.LogFile)
			logger.WithFields(cLog.data).Log(level, message)
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
	if cLog.Type == logType {
		if config.TaskDistrConfig.Enabled {
			if cLog.AllowOnConsole {
				logger.SetOutput(os.Stdout)
				logger.WithFields(cLog.data).Logf(level, msgFmtString, args...)
			}

			logger.SetOutput(cLog.LogFile)
			logger.WithFields(cLog.data).Logf(level, msgFmtString, args...)
		}
	}
	if cLog.next != nil {
		cLog.next.Logf(logType, level, msgFmtString, args...)
	} else {
		// Clearing the fields.
		cLog.resetFields()
	}
}

func (cLog *ClsfnTaskDistrOverheadLogger) CreateLogFile(prefix string) {

	if config.TaskDistrConfig.Enabled {
		filename := strings.Join([]string{prefix, config.TaskDistrConfig.FilenameExtension}, "")
		dirName := logDir.getDirName()
		if dirName != "" {
			if logFile, err := os.Create(filepath.Join(dirName, filename)); err != nil {
				log.Fatal("Unable to create logFile: ", err)
			} else {
				cLog.LogFile = logFile
				cLog.AllowOnConsole = config.TaskDistrConfig.AllowOnConsole
			}
		}
	}
}
