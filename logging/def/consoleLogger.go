package logging

import (
	"log"
)

type ConsoleLogger struct {
	loggerObserverImpl
}

func (cl ConsoleLogger) Log(message string) {
	// We need to log to console only if the message is not empty
	if message != "" {
		log.Println(message)
		// Also logging the message to the console log file
		cl.logObserverSpecifics[conLogger].logFile.Println(message)
	}
}
