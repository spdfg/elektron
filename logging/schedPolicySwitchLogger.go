package logging

import (
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

type schedPolicySwitchLogger struct {
	baseElektronLogger
}

func newSchedPolicySwitchLogger(
	config *loggerConfig,
	b *baseLogData,
	logType int,
	prefix string,
	logger *log.Logger,
	logDir *logDirectory) *schedPolicySwitchLogger {

	sLog := &schedPolicySwitchLogger{
		baseElektronLogger: baseElektronLogger{
			baseLogData: b,
			config: struct {
				Enabled           bool
				FilenameExtension string
				AllowOnConsole    bool
			}{
				Enabled:           config.SPSConfig.Enabled,
				FilenameExtension: config.SPSConfig.FilenameExtension,
				AllowOnConsole:    config.SPSConfig.AllowOnConsole,
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

func (sLog schedPolicySwitchLogger) Log(logType int, level log.Level, message string) {
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
	if sLog.next != nil {
		sLog.next.Log(logType, level, message)
	} else {
		// Clearing the fields.
		sLog.resetFields()
	}
}

func (sLog schedPolicySwitchLogger) Logf(logType int, level log.Level, msgFmtString string, args ...interface{}) {
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
	// Forwarding to next logger
	if sLog.next != nil {
		sLog.next.Logf(logType, level, msgFmtString, args...)
	} else {
		// Clearing the fields.
		sLog.resetFields()
	}
}

func (sLog *schedPolicySwitchLogger) createLogFile(prefix string) {
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
