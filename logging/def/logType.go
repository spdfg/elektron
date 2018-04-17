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
	DEG_COL                 = messageNametoMessageType("DEG_COL")
	SPS                     = messageNametoMessageType("SPS")
	CLSFN_TASKDIST_OVERHEAD = messageNametoMessageType("CLSFN_TASKDIST_OVERHEAD")
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
