package logging

type DegColLogger struct {
	loggerObserverImpl
}

func (pl *DegColLogger) Log(message string) {
	pl.logObserverSpecifics[degColLogger].logFile.Println(message)
}
