package logging

type PCPLogger struct {
	loggerObserverImpl
}

func (pl *PCPLogger) Log(message string) {
	pl.logObserverSpecifics[pcpLogger].logFile.Println(message)
}
