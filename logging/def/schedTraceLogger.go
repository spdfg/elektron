package logging

type SchedTraceLogger struct {
	loggerObserverImpl
}

func (stl SchedTraceLogger) Log(message string) {
	// Logging schedule trace to mentioned file
	stl.logObserverSpecifics[schedTraceLogger].logFile.Println(message)
}
