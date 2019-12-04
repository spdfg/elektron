package elektronLogging

import (
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	. "github.com/spdfg/elektron/elektronLogging/types"
)

var config LoggerConfig
var logger *log.Logger
var formatter ElektronFormatter
var ElektronLogger *LoggerImpl
var logDir logDirectory

func BuildLogger(prefix string, logConfigFilename string) {

	// Read configuration from yaml.
	config.GetConfig(logConfigFilename)

	// Create the log directory.
	startTime := time.Now()
	formatter.TimestampFormat = "2006-01-02 15:04:05"
	formattedStartTime := startTime.Format("20060102150405")
	logDir.createLogDir(prefix, startTime)

	// Instantiate the logrus instance.
	prefix = strings.Join([]string{prefix, formattedStartTime}, "_")
	logger = &log.Logger{
		Out:       os.Stderr,
		Level:     log.DebugLevel,
		Formatter: &formatter,
	}

	// Create a chain of loggers.
	b := &baseLogData{data: log.Fields{}}
	head := &LoggerImpl{baseLogData: b}
	cLog := NewConsoleLogger(b, CONSOLE, prefix)
	pLog := NewPCPLogger(b, PCP, prefix)
	schedTraceLog := NewSchedTraceLogger(b, SCHED_TRACE, prefix)
	spsLog := NewSchedPolicySwitchLogger(b, SPS, prefix)
	schedWindowLog := NewSchedWindowLogger(b, SCHED_WINDOW, prefix)
	tskDistLog := NewClsfnTaskDistrOverheadLogger(b, CLSFN_TASKDISTR_OVERHEAD, prefix)

	head.SetNext(cLog)
	cLog.SetNext(pLog)
	pLog.SetNext(schedTraceLog)
	schedTraceLog.SetNext(spsLog)
	spsLog.SetNext(schedWindowLog)
	schedWindowLog.SetNext(tskDistLog)

	ElektronLogger = head
}
