package elektronLogging

import (
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

type ConsoleLogger struct {
	LoggerImpl
}

func NewConsoleLogger(b *baseLogData, logType int, prefix string, logger *log.Logger) *ConsoleLogger {
	cLog := &ConsoleLogger{}
	cLog.logType = logType
	cLog.CreateLogFile(prefix)
	cLog.next = nil
	cLog.baseLogData = b
	cLog.logger = logger
	return cLog
}
func (cLog ConsoleLogger) Log(logType int, level log.Level, message string) {
	if logType <= cLog.logType {
		if config.ConsoleConfig.Enabled {
			cLog.logger.SetOutput(os.Stdout)
			cLog.logger.WithFields(cLog.data).Log(level, message)

			cLog.logger.SetOutput(cLog.logFile)
			cLog.logger.WithFields(cLog.data).Log(level, message)
		}
	}
	// Forwarding to next logger.
	if cLog.next != nil {
		cLog.next.Log(logType, level, message)
	} else {
		// Clearing the fields.
		cLog.resetFields()
	}
}

func (cLog ConsoleLogger) Logf(logType int, level log.Level, msgFmtString string, args ...interface{}) {
	if logType <= cLog.logType {
		if config.ConsoleConfig.Enabled {
			cLog.logger.SetOutput(os.Stdout)
			cLog.logger.WithFields(cLog.data).Logf(level, msgFmtString, args...)

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

func (cLog *ConsoleLogger) CreateLogFile(prefix string) {
	// Create log file for the type if it is enabled.
	if config.ConsoleConfig.Enabled {
		filename := strings.Join([]string{prefix, config.ConsoleConfig.FilenameExtension}, "")
		dirName := logDir.getDirName()
		if dirName != "" {
			if logFile, err := os.Create(filepath.Join(dirName, filename)); err != nil {
				log.Fatal("Unable to create logFile: ", err)
			} else {
				cLog.logFile = logFile
				cLog.allowOnConsole = config.ConsoleConfig.AllowOnConsole
			}
		}
	}
}
