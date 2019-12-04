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

func NewConsoleLogger(b *baseLogData, logType int, prefix string) *ConsoleLogger {
	cLog := &ConsoleLogger{}
	cLog.Type = logType
	cLog.CreateLogFile(prefix)
	cLog.next = nil
	cLog.baseLogData = b
	return cLog
}
func (cLog ConsoleLogger) Log(logType int, level log.Level, message string) {
	if config.ConsoleConfig.Enabled {
		if logType <= cLog.Type {

			logger.SetOutput(os.Stdout)
			logger.WithFields(cLog.data).Log(level, message)

			logger.SetOutput(cLog.LogFile)
			logger.WithFields(cLog.data).Log(level, message)
		}
		if cLog.next != nil {
			cLog.next.Log(logType, level, message)
		} else {
			// Clearing the fields.
			cLog.resetFields()
		}
	}
}

func (cLog ConsoleLogger) Logf(logType int, level log.Level, msgFmtString string, args ...interface{}) {
	if config.ConsoleConfig.Enabled {
		if logType <= cLog.Type {

			logger.SetOutput(os.Stdout)
			logger.WithFields(cLog.data).Logf(level, msgFmtString, args...)

			logger.SetOutput(cLog.LogFile)
			logger.WithFields(cLog.data).Logf(level, msgFmtString, args...)
		}
		if cLog.next != nil {
			cLog.next.Logf(logType, level, msgFmtString, args...)
		} else {
			// Clearing the fields.
			cLog.resetFields()
		}
	}
}

func (cLog *ConsoleLogger) CreateLogFile(prefix string) {

	if config.ConsoleConfig.Enabled {
		filename := strings.Join([]string{prefix, config.ConsoleConfig.FilenameExtension}, "")
		dirName := logDir.getDirName()
		if dirName != "" {
			if logFile, err := os.Create(filepath.Join(dirName, filename)); err != nil {
				log.Fatal("Unable to create logFile: ", err)
			} else {
				cLog.LogFile = logFile
				cLog.AllowOnConsole = config.ConsoleConfig.AllowOnConsole
			}
		}
	}
}
