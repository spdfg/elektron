package logging

import (
	"fmt"
	"log"
	"os"
)

// Logging platform
type loggerObserver interface {
	Log(message string)
	setLogFile()
	setLogFilePrefix(prefix string)
	setLogDirectory(dirName string)
	init(opts ...loggerOption)
}

type specifics struct {
	logFilePrefix string
	logFile       *log.Logger
}

type loggerObserverImpl struct {
	logFile              *log.Logger
	logObserverSpecifics map[string]*specifics
	logDirectory         string
}

func (loi *loggerObserverImpl) init(opts ...loggerOption) {
	for _, opt := range opts {
		// applying logger options
		if err := opt(loi); err != nil {
			log.Fatal(err)
		}
	}
}

func (loi loggerObserverImpl) Log(message string) {}

// Requires logFilePrefix to have already been set
func (loi *loggerObserverImpl) setLogFile() {
	for prefix, ls := range loi.logObserverSpecifics {
		if logFile, err := os.Create(ls.logFilePrefix); err != nil {
			log.Fatal("Unable to create logFile: ", err)
		} else {
			fmt.Printf("Creating logFile with pathname: %s, and prefix: %s\n", ls.logFilePrefix, prefix)
			ls.logFile = log.New(logFile, "", log.LstdFlags)
		}
	}
}

func (loi *loggerObserverImpl) setLogFilePrefix(prefix string) {
	// Setting logFilePrefix for pcp logger
	pcpLogFilePrefix := prefix + ".pcplog"
	if loi.logDirectory != "" {
		pcpLogFilePrefix = loi.logDirectory + "/" + pcpLogFilePrefix
	}
	loi.logObserverSpecifics[pcpLogger].logFilePrefix = pcpLogFilePrefix

	// Setting logFilePrefix for console logger
	consoleLogFilePrefix := prefix + "_console.log"
	if loi.logDirectory != "" {
		consoleLogFilePrefix = loi.logDirectory + "/" + consoleLogFilePrefix
	}
	loi.logObserverSpecifics[conLogger].logFilePrefix = consoleLogFilePrefix

	// Setting logFilePrefix for schedTrace logger
	schedTraceLogFilePrefix := prefix + "_schedTrace.log"
	if loi.logDirectory != "" {
		schedTraceLogFilePrefix = loi.logDirectory + "/" + schedTraceLogFilePrefix
	}
	loi.logObserverSpecifics[schedTraceLogger].logFilePrefix = schedTraceLogFilePrefix

	// Setting logFilePrefix for degCol logger
	degColLogFilePrefix := prefix + "_degCol.log"
	if loi.logDirectory != "" {
		schedTraceLogFilePrefix = loi.logDirectory + "/" + degColLogFilePrefix
	}
	loi.logObserverSpecifics[degColLogger].logFilePrefix = degColLogFilePrefix
}

func (loi *loggerObserverImpl) setLogDirectory(dirName string) {
	loi.logDirectory = dirName
}
