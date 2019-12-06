package logging

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"strings"
)

type consoleLogger struct {
	baseElektronLogger
	MinLogLevel string
}

func newConsoleLogger(
	config *loggerConfig,
	b *baseLogData,
	logType int,
	prefix string,
	logger *log.Logger,
	logDir *logDirectory) *consoleLogger {

	cLog := &consoleLogger{
		baseElektronLogger: baseElektronLogger{
			baseLogData: b,
			config: struct {
				Enabled           bool
				FilenameExtension string
				AllowOnConsole    bool
			}{
				Enabled:           config.ConsoleConfig.Enabled,
				FilenameExtension: config.ConsoleConfig.FilenameExtension,
				AllowOnConsole:    config.ConsoleConfig.AllowOnConsole,
			},
			logType: logType,
			next:    nil,
			logger:  logger,
			logDir:  logDir,
		},

		MinLogLevel: config.ConsoleConfig.MinLogLevel,
	}

	cLog.createLogFile(prefix)
	return cLog
}

func (cLog consoleLogger) Log(logType int, level log.Level, message string) {
	if logType <= cLog.logType {
		if cLog.isEnabled() {
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

func (cLog consoleLogger) Logf(logType int, level log.Level, msgFmtString string, args ...interface{}) {
	if logType <= cLog.logType {
		if cLog.isEnabled() {
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

func (cLog *consoleLogger) createLogFile(prefix string) {
	// Create log file for the type if it is enabled.
	if cLog.isEnabled() {
		filename := strings.Join([]string{prefix, cLog.config.FilenameExtension}, "")
		dirName := cLog.logDir.getDirName()
		fmt.Println(dirName)
		if dirName != "" {
			if logFile, err := os.Create(filepath.Join(dirName, filename)); err != nil {
				log.Fatal("Unable to create logFile: ", err)
			} else {
				cLog.logFile = logFile
			}
		}
	}
}
