package logging

import (
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

type schedWindowLogger struct {
	baseElektronLogger
}

func newSchedWindowLogger(
	config *loggerConfig,
	b *baseLogData,
	logType int,
	prefix string,
	logger *log.Logger,
	logDir *logDirectory) *schedWindowLogger {

	sLog := &schedWindowLogger{
		baseElektronLogger: baseElektronLogger{
			baseLogData: b,
			config: struct {
				Enabled           bool
				FilenameExtension string
				AllowOnConsole    bool
			}{
				Enabled:           config.SchedWindowConfig.Enabled,
				FilenameExtension: config.SchedWindowConfig.FilenameExtension,
				AllowOnConsole:    config.SchedWindowConfig.AllowOnConsole,
			},
			logType: logType,
			next:    nil,
			logger:  logger,
			logDir:  logDir,
		},
	}

	sLog.createLogFile(prefix)
	return sLog
}

func (sLog schedWindowLogger) Log(logType int, level log.Level, message string) {
	if sLog.logType == logType {
		if sLog.isEnabled() {
			if sLog.isAllowedOnConsole() {
				sLog.logger.SetOutput(os.Stdout)
				sLog.logger.WithFields(sLog.data).Log(level, message)
			}

			sLog.logger.SetOutput(sLog.logFile)
			sLog.logger.WithFields(sLog.data).Log(level, message)
		}
	}
	// Forwarding to next logger
	if sLog.next != nil {
		sLog.next.Log(logType, level, message)
	} else {
		// Clearing the fields.
		sLog.resetFields()
	}
}

func (sLog schedWindowLogger) Logf(logType int, level log.Level, msgFmtString string, args ...interface{}) {
	if sLog.logType == logType {
		if sLog.isEnabled() {
			if sLog.isAllowedOnConsole() {
				sLog.logger.SetOutput(os.Stdout)
				sLog.logger.WithFields(sLog.data).Logf(level, msgFmtString, args...)
			}

			sLog.logger.SetOutput(sLog.logFile)
			sLog.logger.WithFields(sLog.data).Logf(level, msgFmtString, args...)
		}
	}
	if sLog.next != nil {
		sLog.next.Logf(logType, level, msgFmtString, args...)
	} else {
		// Clearing the fields.
		sLog.resetFields()
	}
}

func (sLog *schedWindowLogger) createLogFile(prefix string) {
	if sLog.isEnabled() {
		filename := strings.Join([]string{prefix, sLog.getFilenameExtension()}, "")
		dirName := sLog.logDir.getDirName()
		if dirName != "" {
			if logFile, err := os.Create(filepath.Join(dirName, filename)); err != nil {
				log.Fatal("Unable to create logFile: ", err)
			} else {
				sLog.logFile = logFile
			}
		}
	}
}
