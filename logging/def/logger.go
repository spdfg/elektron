// Copyright (C) 2018 spdfg
// 
// This file is part of Elektron.
// 
// Elektron is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
// 
// Elektron is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
// 
// You should have received a copy of the GNU General Public License
// along with Elektron.  If not, see <http://www.gnu.org/licenses/>.
// 

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
			ERROR:                   true,
			GENERAL:                 true,
			WARNING:                 true,
			SCHED_TRACE:             true,
			SUCCESS:                 true,
			PCP:                     true,
			SPS:                     true,
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
