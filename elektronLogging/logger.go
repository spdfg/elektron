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
var ElektronLogger *ConsoleLogger
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
	cLog := NewConsoleLogger(CONSOLE, prefix)
	pLog := NewPCPLogger(PCP, prefix)
	schedTraceLog := NewSchedTraceLogger(SCHED_TRACE, prefix)
	spsLog := NewSchedPolicySwitchLogger(SPS, prefix)
	schedWindowLog := NewSchedWindowLogger(SCHED_WINDOW, prefix)
	tskDistLog := NewClsfnTaskDistrOverheadLogger(CLSFN_TASKDIST_OVERHEAD, prefix)

	cLog.SetNext(pLog)
	pLog.SetNext(schedTraceLog)
	schedTraceLog.SetNext(spsLog)
	spsLog.SetNext(schedWindowLog)
	schedWindowLog.SetNext(tskDistLog)

	ElektronLogger = cLog
}
