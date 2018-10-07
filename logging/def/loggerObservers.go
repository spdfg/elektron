// Copyright (C) 2018 spdf
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
	"fmt"
	"log"
	"os"
)

// Logging platform
type loggerObserver interface {
	Log(message string)
	setLogFile()
	setLogFilePrefix(prefix string)
	setLogDirectory(dirName string)
	init(opts ...loggerOption)
}

type specifics struct {
	logFilePrefix string
	logFile       *log.Logger
}

type loggerObserverImpl struct {
	logFile              *log.Logger
	logObserverSpecifics map[string]*specifics
	logDirectory         string
}

func (loi *loggerObserverImpl) init(opts ...loggerOption) {
	for _, opt := range opts {
		// applying logger options
		if err := opt(loi); err != nil {
			log.Fatal(err)
		}
	}
}

func (loi loggerObserverImpl) Log(message string) {}

// Requires logFilePrefix to have already been set
func (loi *loggerObserverImpl) setLogFile() {
	for prefix, ls := range loi.logObserverSpecifics {
		if logFile, err := os.Create(ls.logFilePrefix); err != nil {
			log.Fatal("Unable to create logFile: ", err)
		} else {
			fmt.Printf("Creating logFile with pathname: %s, and prefix: %s\n", ls.logFilePrefix, prefix)
			ls.logFile = log.New(logFile, "", log.LstdFlags)
		}
	}
}

func (loi *loggerObserverImpl) setLogFilePrefix(prefix string) {
	// Setting logFilePrefix for pcp logger
	pcpLogFilePrefix := prefix + ".pcplog"
	if loi.logDirectory != "" {
		pcpLogFilePrefix = loi.logDirectory + "/" + pcpLogFilePrefix
	}
	loi.logObserverSpecifics[pcpLogger].logFilePrefix = pcpLogFilePrefix

	// Setting logFilePrefix for console logger
	consoleLogFilePrefix := prefix + "_console.log"
	if loi.logDirectory != "" {
		consoleLogFilePrefix = loi.logDirectory + "/" + consoleLogFilePrefix
	}
	loi.logObserverSpecifics[conLogger].logFilePrefix = consoleLogFilePrefix

	// Setting logFilePrefix for schedTrace logger
	schedTraceLogFilePrefix := prefix + "_schedTrace.log"
	if loi.logDirectory != "" {
		schedTraceLogFilePrefix = loi.logDirectory + "/" + schedTraceLogFilePrefix
	}
	loi.logObserverSpecifics[schedTraceLogger].logFilePrefix = schedTraceLogFilePrefix

	// Setting logFilePrefix for schedulingPolicySwitch logger
	schedPolicySwitchLogFilePrefix := prefix + "_schedPolicySwitch.log"
	if loi.logDirectory != "" {
		schedPolicySwitchLogFilePrefix = loi.logDirectory + "/" + schedPolicySwitchLogFilePrefix
	}
	loi.logObserverSpecifics[spsLogger].logFilePrefix = schedPolicySwitchLogFilePrefix

	// Setting logFilePrefix for clsfnTaskDist logger.
	// Execution time of every call to def.GetTaskDistribution(...) would be recorded and logged in this file.
	// The overhead would be logged in microseconds.
	clsfnTaskDistOverheadLogFilePrefix := prefix + "_classificationOverhead.log"
	if loi.logDirectory != "" {
		clsfnTaskDistOverheadLogFilePrefix = loi.logDirectory + "/" + clsfnTaskDistOverheadLogFilePrefix
	}
	loi.logObserverSpecifics[clsfnTaskDistOverheadLogger].logFilePrefix = clsfnTaskDistOverheadLogFilePrefix

	// Setting logFilePrefix for schedWindow logger.
	// Going to log the time stamp when the scheduling window was determined
	// 	and the size of the scheduling window.
	schedWindowLogFilePrefix := prefix + "_schedWindow.log"
	if loi.logDirectory != "" {
		schedWindowLogFilePrefix = loi.logDirectory + "/" + schedWindowLogFilePrefix
	}
	loi.logObserverSpecifics[schedWindowLogger].logFilePrefix = schedWindowLogFilePrefix
}

func (loi *loggerObserverImpl) setLogDirectory(dirName string) {
	loi.logDirectory = dirName
}
