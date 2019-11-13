package elektronLogging

import (
	"time"
    "fmt"
    "strconv"
	. "github.com/spdfg/elektron/elektronLogging/types"
	log "github.com/sirupsen/logrus"
)

var config LoggerConfig
var logger *log.Logger
var formatter ElektronFormatter

func BuildLogger() *LoggerImpl {

    // read configuration from yaml
	config.GetConfig()
    
    // create the log directory
	startTime := time.Now()
	formatter.TimestampFormat = "2006-01-02 15:04:05"
	GetLogDir(startTime, "_")

    prefix := fmt.Sprintf("_%d%d%s%s%s%s",startTime.Year(), startTime.Month(),strconv.Itoa(startTime.Day()),
                strconv.Itoa(startTime.Hour()),strconv.Itoa(startTime.Minute()),strconv.Itoa(startTime.Second()))
	
    //create a single logrus instance and set its formatter to ElektronFormatter
	logger = log.New()
	logger.SetFormatter(&formatter)

    // create a chain of loggers
	head := new(LoggerImpl)
	cLog := NewConsoleLogger(CONSOLE,prefix)
	pLog := NewPcpLogger(PCP,prefix)
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

	return head
}

var ElektronLog = BuildLogger()
