package logging

type loggerSubject struct {
	Registry map[LogMessageType][]loggerObserver
	message  string
}

func (ls *loggerSubject) setMessage(message string) {
	ls.message = message
}

func (ls *loggerSubject) attach(messageType LogMessageType, lo loggerObserver) {
	if ls.Registry == nil {
		ls.Registry = make(map[LogMessageType][]loggerObserver)
	}
	ls.Registry[messageType] = append(ls.Registry[messageType], lo)
}

func (ls *loggerSubject) notify(messageType LogMessageType) {
	for _, logObserver := range ls.Registry[messageType] {
		logObserver.Log(ls.message)
	}
}
