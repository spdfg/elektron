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
//var logDir string

func BuildLogger() *LoggerImpl {

	config.GetConfig()
	startTime := time.Now()
	formatter.TimestampFormat = "2006-01-02 15:04:05"
	GetLogDir(startTime, "_")

    prefix := fmt.Sprintf("_%s%s%s%s%s",startTime.Month().String(),strconv.Itoa(startTime.Day()),
                strconv.Itoa(startTime.Hour()),strconv.Itoa(startTime.Minute()),strconv.Itoa(startTime.Second()))
	
	logger = log.New()
	logger.SetFormatter(&formatter)
	
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
