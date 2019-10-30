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

import "github.com/fatih/color"

// Defining enums of log message types
var logMessageNames []string

// Possible log message types
var (
	ERROR                   = messageNametoMessageType("ERROR")
	WARNING                 = messageNametoMessageType("WARNING")
	GENERAL                 = messageNametoMessageType("GENERAL")
	SUCCESS                 = messageNametoMessageType("SUCCESS")
	SCHED_TRACE             = messageNametoMessageType("SCHED_TRACE")
	PCP                     = messageNametoMessageType("PCP")
	SPS                     = messageNametoMessageType("SPS")
	CLSFN_TASKDIST_OVERHEAD = messageNametoMessageType("CLSFN_TASKDIST_OVERHEAD")
	SCHED_WINDOW            = messageNametoMessageType("SCHED_WINDOW")
)

// Text colors for the different types of log messages.
var LogMessageColors map[LogMessageType]*color.Color = map[LogMessageType]*color.Color{
	ERROR:   color.New(color.FgRed, color.Bold),
	WARNING: color.New(color.FgYellow, color.Bold),
	GENERAL: color.New(color.FgWhite, color.Bold),
	SUCCESS: color.New(color.FgGreen, color.Bold),
}

type LogMessageType int

func (lmt LogMessageType) String() string {
	return logMessageNames[lmt]
}

func GetLogMessageTypes() []string {
	return logMessageNames
}

func messageNametoMessageType(messageName string) LogMessageType {
	// Appending messageName to LogMessageNames
	logMessageNames = append(logMessageNames, messageName)
	// Mapping messageName to int
	return LogMessageType(len(logMessageNames) - 1)
}
