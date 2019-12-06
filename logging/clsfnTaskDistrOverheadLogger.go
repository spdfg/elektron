package logging

import (
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

type clsfnTaskDistrOverheadLogger struct {
	baseElektronLogger
}

func newClsfnTaskDistrOverheadLogger(
	config *loggerConfig,
	b *baseLogData,
	logType int,
	prefix string,
	logger *log.Logger,
	logDir *logDirectory) *clsfnTaskDistrOverheadLogger {

	cLog := &clsfnTaskDistrOverheadLogger{
		baseElektronLogger: baseElektronLogger{
			baseLogData: b,
			config: struct {
				Enabled           bool
				FilenameExtension string
				AllowOnConsole    bool
			}{
				Enabled:           config.TaskDistrConfig.Enabled,
				FilenameExtension: config.TaskDistrConfig.FilenameExtension,
				AllowOnConsole:    config.TaskDistrConfig.AllowOnConsole,
			},
			logType: logType,
			next:    nil,
			logger:  logger,
			logDir:  logDir,
		},
	}

	cLog.createLogFile(prefix)
	return cLog
}
func (cLog clsfnTaskDistrOverheadLogger) Log(logType int, level log.Level, message string) {
	if cLog.logType == logType {
		if cLog.isEnabled() {
			if cLog.isAllowedOnConsole() {
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

func (cLog clsfnTaskDistrOverheadLogger) Logf(logType int, level log.Level, msgFmtString string, args ...interface{}) {
	if cLog.logType == logType {
		if cLog.isEnabled() {
			if cLog.isAllowedOnConsole() {
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

func (cLog *clsfnTaskDistrOverheadLogger) createLogFile(prefix string) {

	if cLog.isEnabled() {
		filename := strings.Join([]string{prefix, cLog.getFilenameExtension()}, "")
		dirName := cLog.logDir.getDirName()
		if dirName != "" {
			if logFile, err := os.Create(filepath.Join(dirName, filename)); err != nil {
				log.Fatal("Unable to create logFile: ", err)
			} else {
				cLog.logFile = logFile
			}
		}
	}
}
