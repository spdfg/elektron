package logging

import (
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

type pcpLogger struct {
	baseElektronLogger
}

func newPCPLogger(
	config *loggerConfig,
	b *baseLogData,
	logType int,
	prefix string,
	logger *log.Logger,
	logDir *logDirectory) *pcpLogger {

	pLog := &pcpLogger{
		baseElektronLogger: baseElektronLogger{
			baseLogData: b,
			config: struct {
				Enabled           bool
				FilenameExtension string
				AllowOnConsole    bool
			}{
				Enabled:           config.PCPConfig.Enabled,
				FilenameExtension: config.PCPConfig.FilenameExtension,
				AllowOnConsole:    config.PCPConfig.AllowOnConsole,
			},
			logType: logType,
			next:    nil,
			logger:  logger,
			logDir:  logDir,
		},
	}

	pLog.createLogFile(prefix)
	return pLog
}

func (pLog pcpLogger) Log(logType int, level log.Level, message string) {
	if pLog.logType == logType {
		if pLog.isEnabled() {
			if pLog.isAllowedOnConsole() {
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

func (pLog pcpLogger) Logf(logType int, level log.Level, msgFmtString string, args ...interface{}) {
	if pLog.logType == logType {
		if pLog.isEnabled() {
			if pLog.isAllowedOnConsole() {
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

func (pLog *pcpLogger) createLogFile(prefix string) {
	if pLog.isEnabled() {
		filename := strings.Join([]string{prefix, pLog.getFilenameExtension()}, "")
		dirName := pLog.logDir.getDirName()
		if dirName != "" {
			if logFile, err := os.Create(filepath.Join(dirName, filename)); err != nil {
				log.Fatal("Unable to create logFile: ", err)
			} else {
				pLog.logFile = logFile
			}
		}
	}
}
