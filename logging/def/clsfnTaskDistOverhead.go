package logging

type ClsfnTaskDistOverheadLogger struct {
	loggerObserverImpl
}

func (col ClsfnTaskDistOverheadLogger) Log(message string) {
	// Logging the overhead of classifying tasks in the scheduling window and determining the distribution
	//      of light power consuming and heavy power consuming tasks.
	col.logObserverSpecifics[clsfnTaskDistOverheadLogger].logFile.Println(message)
}
