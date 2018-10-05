package logging

import (
	"strings"
	"time"

	logUtils "gitlab.com/spdf/elektron/logging/utils"
)

// Names of different loggers
const (
	conLogger                   = "console-logger"
	schedTraceLogger            = "schedTrace-logger"
	pcpLogger                   = "pcp-logger"
	spsLogger                   = "schedPolicySwitch-logger"
	clsfnTaskDistOverheadLogger = "classificationOverhead-logger"
	schedWindowLogger           = "schedWindow-logger"
)

// Logger class factory
var Loggers map[string]loggerObserver = map[string]loggerObserver{
	conLogger:                   nil,
	schedTraceLogger:            nil,
	pcpLogger:                   nil,
	spsLogger:                   nil,
	clsfnTaskDistOverheadLogger: nil,
	schedWindowLogger:           nil,
}

// Logger options to help initialize loggers
type loggerOption func(l loggerObserver) error

func withLogDirectory(startTime time.Time, prefix string) loggerOption {
	return func(l loggerObserver) error {
		l.(*loggerObserverImpl).setLogDirectory(logUtils.GetLogDir(startTime, prefix))
		return nil
	}
}

// This loggerOption initializes the specifics for each loggerObserver
func withLoggerSpecifics(prefix string) loggerOption {
	return func(l loggerObserver) error {
		l.(*loggerObserverImpl).logObserverSpecifics = map[string]*specifics{
			conLogger:                   &specifics{},
			schedTraceLogger:            &specifics{},
			pcpLogger:                   &specifics{},
			spsLogger:                   &specifics{},
			clsfnTaskDistOverheadLogger: &specifics{},
			schedWindowLogger:           &specifics{},
		}
		l.(*loggerObserverImpl).setLogFilePrefix(prefix)
		l.(*loggerObserverImpl).setLogFile()
		return nil
	}
}

// Build and assign all loggers
func attachAllLoggers(lg *LoggerDriver, startTime time.Time, prefix string) {
	loi := &loggerObserverImpl{}
	loi.init(withLogDirectory(startTime, strings.Split(prefix, startTime.Format("20060102150405"))[0]),
		withLoggerSpecifics(prefix))
	Loggers[conLogger] = &ConsoleLogger{
		loggerObserverImpl: *loi,
	}
	Loggers[schedTraceLogger] = &SchedTraceLogger{
		loggerObserverImpl: *loi,
	}
	Loggers[pcpLogger] = &PCPLogger{
		loggerObserverImpl: *loi,
	}
	Loggers[spsLogger] = &SchedPolicySwitchLogger{
		loggerObserverImpl: *loi,
	}
	Loggers[clsfnTaskDistOverheadLogger] = &ClsfnTaskDistOverheadLogger{
		loggerObserverImpl: *loi,
	}
	Loggers[schedWindowLogger] = &SchedWindowLogger{
		loggerObserverImpl: *loi,
	}

	for _, lmt := range GetLogMessageTypes() {
		switch lmt {
		case SCHED_TRACE.String():
			lg.attach(SCHED_TRACE, Loggers[schedTraceLogger])
		case GENERAL.String():
			lg.attach(GENERAL, Loggers[conLogger])
		case WARNING.String():
			lg.attach(WARNING, Loggers[conLogger])
		case ERROR.String():
			lg.attach(ERROR, Loggers[conLogger])
		case SUCCESS.String():
			lg.attach(SUCCESS, Loggers[conLogger])
		case PCP.String():
			lg.attach(PCP, Loggers[pcpLogger])
		case SPS.String():
			lg.attach(SPS, Loggers[spsLogger])
		case CLSFN_TASKDIST_OVERHEAD.String():
			lg.attach(CLSFN_TASKDIST_OVERHEAD, Loggers[clsfnTaskDistOverheadLogger])
		case SCHED_WINDOW.String():
			lg.attach(SCHED_WINDOW, Loggers[schedWindowLogger])
		}
	}
}
