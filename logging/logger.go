package logging

import (
	"github.com/pkg/errors"
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	. "github.com/spdfg/elektron/logging/types"
)

// var config loggerConfig
var formatter elektronFormatter
var elektronLoggerInstance elektronLogger

type elektronLogger interface {
	setNext(next elektronLogger)
	Log(logType int, level log.Level, message string)
	Logf(logType int, level log.Level, msgFmtString string, args ...interface{})
	WithFields(logData log.Fields) elektronLogger
	WithField(key string, value string) elektronLogger
}

type baseLogData struct {
	data log.Fields
}

type baseElektronLogger struct {
	*baseLogData

	config struct {
		Enabled           bool
		FilenameExtension string
		AllowOnConsole    bool
	}

	logType int
	logFile *os.File
	next    elektronLogger
	logger  *log.Logger
	logDir  *logDirectory
}

func (l baseElektronLogger) isEnabled() bool {
	return l.config.Enabled
}

func (l baseElektronLogger) isAllowedOnConsole() bool {
	return l.config.AllowOnConsole
}

func (l baseElektronLogger) getFilenameExtension() string {
	return l.config.FilenameExtension
}

func (l *baseElektronLogger) WithFields(logData log.Fields) elektronLogger {
	l.data = logData
	return l
}

func (l *baseElektronLogger) WithField(key string, value string) elektronLogger {
	l.data[key] = value
	return l
}

func (l *baseElektronLogger) setNext(next elektronLogger) {
	l.next = next
}

func (l baseElektronLogger) Log(logType int, level log.Level, message string) {
	if l.next != nil {
		l.next.Log(logType, level, message)
	}
}

func (l baseElektronLogger) Logf(logType int, level log.Level, msgFmtString string, args ...interface{}) {
	if l.next != nil {
		l.next.Logf(logType, level, msgFmtString, args...)
	}
}

func (l *baseElektronLogger) resetFields() {
	l.data = nil
	l.data = log.Fields{}
}

func BuildLogger(prefix string, logConfigFilename string) error {

	// Create the log directory.
	startTime := time.Now()
	formatter.TimestampFormat = "2006-01-02 15:04:05"
	formattedStartTime := startTime.Format("20060102150405")
	logDir := &logDirectory{}
	logDir.createLogDir(prefix, startTime)

	// Instantiate the logrus instance.
	prefix = strings.Join([]string{prefix, formattedStartTime}, "_")
	logger := &log.Logger{
		Out:       os.Stderr,
		Level:     log.DebugLevel,
		Formatter: &formatter,
	}

	// Create a chain of loggers.
	b := &baseLogData{data: log.Fields{}}
	head := &baseElektronLogger{baseLogData: b}

	// Read configuration from yaml.
	if config, err := GetConfig(logConfigFilename); err != nil {
		return errors.Wrap(err, "Failed to build logger")
	} else {
		cLog := newConsoleLogger(config, b, CONSOLE, prefix, logger, logDir)
		pLog := newPCPLogger(config, b, PCP, prefix, logger, logDir)
		schedTraceLog := newSchedTraceLogger(config, b, SCHED_TRACE, prefix, logger, logDir)
		spsLog := newSchedPolicySwitchLogger(config, b, SPS, prefix, logger, logDir)
		schedWindowLog := newSchedWindowLogger(config, b, SCHED_WINDOW, prefix, logger, logDir)
		tskDistLog := newClsfnTaskDistrOverheadLogger(config, b, CLSFN_TASKDISTR_OVERHEAD, prefix, logger, logDir)

		head.setNext(cLog)
		cLog.setNext(pLog)
		pLog.setNext(schedTraceLog)
		schedTraceLog.setNext(spsLog)
		spsLog.setNext(schedWindowLog)
		schedWindowLog.setNext(tskDistLog)

	}

	elektronLoggerInstance = head
	return nil
}

func Log(logType int, level log.Level, message string) {
	elektronLoggerInstance.Log(logType, level, message)
}

func Logf(logType int, level log.Level, msgFmtString string, args ...interface{}) {
	elektronLoggerInstance.Logf(logType, level, msgFmtString, args...)
}

func WithFields(logData log.Fields) elektronLogger {
	return elektronLoggerInstance.WithFields(logData)
}

func WithField(key string, value string) elektronLogger {
	return elektronLoggerInstance.WithField(key, value)
}
