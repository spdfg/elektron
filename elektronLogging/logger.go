package elektronLogging

import (
	log "github.com/sirupsen/logrus"
	. "github.com/spdfg/elektron/elektronLogging/types"
	"os"
	"strings"
	"time"
)

var config LoggerConfig
var logger *log.Logger
var formatter ElektronFormatter
var ElektronLog *LoggerImpl

func BuildLogger(prefix string) {

	// read configuration from yaml
	config.GetConfig()

	// create the log directory
	startTime := time.Now()
	formatter.TimestampFormat = "2006-01-02 15:04:05"
	formattedStartTime := startTime.Format("20060102150405")
	GetLogDir(startTime, "_")

	prefix = strings.Join([]string{prefix, formattedStartTime}, "_")

	logger = &log.Logger{
		Out:       os.Stderr,
		Level:     log.DebugLevel,
		Formatter: &formatter,
	}

	// create a chain of loggers
	head := &LoggerImpl{}
	cLog := NewConsoleLogger(CONSOLE, prefix)
	pLog := NewPcpLogger(PCP, prefix)
	schedTraceLog := NewSchedTraceLogger(SCHED_TRACE, prefix)
	spsLog := NewSchedPolicySwitchLogger(SPS, prefix)
	schedWindowLog := NewSchedWindowLogger(SCHED_WINDOW, prefix)
	tskDistLog := NewClsfnTaskDistOverheadLogger(CLSFN_TASKDIST_OVERHEAD, prefix)

	head.SetNext(cLog)
	cLog.SetNext(pLog)
	pLog.SetNext(schedTraceLog)
	schedTraceLog.SetNext(spsLog)
	spsLog.SetNext(schedWindowLog)
	schedWindowLog.SetNext(tskDistLog)

	ElektronLog = head
}
