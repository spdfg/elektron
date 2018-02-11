package logging

type SchedPolicySwitchLogger struct {
	loggerObserverImpl
}

func (pl *SchedPolicySwitchLogger) Log(message string) {
	pl.logObserverSpecifics[spsLogger].logFile.Println(message)
}
