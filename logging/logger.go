package logging

import (
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	. "github.com/spdfg/elektron/logging/types"
)

var config LoggerConfig
var formatter ElektronFormatter
var ElektronLogger *loggerImpl

func BuildLogger(prefix string, logConfigFilename string) {

	// Read configuration from yaml.
	config.GetConfig(logConfigFilename)

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
	head := &loggerImpl{baseLogData: b}
	cLog := NewConsoleLogger(b, CONSOLE, prefix, logger, logDir)
	pLog := NewPCPLogger(b, PCP, prefix, logger, logDir)
	schedTraceLog := NewSchedTraceLogger(b, SCHED_TRACE, prefix, logger, logDir)
	spsLog := NewSchedPolicySwitchLogger(b, SPS, prefix, logger, logDir)
	schedWindowLog := NewSchedWindowLogger(b, SCHED_WINDOW, prefix, logger, logDir)
	tskDistLog := NewClsfnTaskDistrOverheadLogger(b, CLSFN_TASKDISTR_OVERHEAD, prefix, logger, logDir)

	head.setNext(cLog)
	cLog.setNext(pLog)
	pLog.setNext(schedTraceLog)
	schedTraceLog.setNext(spsLog)
	spsLog.setNext(schedWindowLog)
	schedWindowLog.setNext(tskDistLog)

	ElektronLogger = head
}
