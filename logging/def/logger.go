package logging

import (
	"time"
)

type LoggerDriver struct {
	loggerSubject
	allowedMessageTypes map[LogMessageType]bool
}

func newLogger() *LoggerDriver {
	logger := &LoggerDriver{
		allowedMessageTypes: map[LogMessageType]bool{
			ERROR:       true,
			GENERAL:     true,
			WARNING:     true,
			SCHED_TRACE: true,
			SUCCESS:     true,
			PCP:         true,
			SPS:         true,
			CLSFN_TASKDIST_OVERHEAD: true,
			SCHED_WINDOW:            true,
		},
	}
	return logger
}

func BuildLogger(startTime time.Time, prefix string) *LoggerDriver {
	// building logger
	l := newLogger()
	attachAllLoggers(l, startTime, prefix)
	return l
}

func (log *LoggerDriver) EnabledLogging(messageType LogMessageType) {
	log.allowedMessageTypes[messageType] = true
}

func (log *LoggerDriver) DisableLogging(messageType LogMessageType) {
	log.allowedMessageTypes[messageType] = false
}

func (log *LoggerDriver) WriteLog(messageType LogMessageType, message string) {
	// checking to see if logging for given messageType is disabled
	if log.allowedMessageTypes[messageType] {
		log.setMessage(message)
		// notify registered loggers to log
		log.notify(messageType)
	}
}

func (log *LoggerDriver) Listen(logMType <-chan LogMessageType, logMsg <-chan string) {
	for {
		mType := <-logMType
		msg := <-logMsg
		log.WriteLog(mType, msg)
	}
}
