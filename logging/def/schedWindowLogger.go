package logging

type SchedWindowLogger struct {
	loggerObserverImpl
}

func (swl SchedWindowLogger) Log(message string) {
	// Logging schedule trace to mentioned file
	swl.logObserverSpecifics[schedWindowLogger].logFile.Println(message)
}
