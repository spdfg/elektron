package elektronLogging

import (
	"os"
	log "github.com/sirupsen/logrus"
	//data "github.com/spdfg/elektron/elektronLogging/data"
)

type ConsoleLogger struct {
	LoggerImpl
}

func NewConsoleLogger(logType int, prefix string) *ConsoleLogger {
	cLog := new(ConsoleLogger)
	cLog.Type = logType
	cLog.SetLogFile(prefix)
	return cLog
}
func (cLog *ConsoleLogger) Log(logType int, level log.Level, logData log.Fields, message string) {
	if logType <= cLog.Type {

		//logFields := cloneFields(logData)
		log.SetLevel(level)
		
		log.SetOutput(os.Stdout)
		log.WithFields(logData).Println(message)
		
		log.SetOutput(cLog.LogFileName)
		log.WithFields(logData).Println(message)
	}
	if cLog.next != nil {
		cLog.next.Log(logType, level, logData, message)
	}
}

func (cLog *ConsoleLogger) SetLogFile(prefix string) {

	consoleLogPrefix := prefix + config.ConsoleConfig.FilenameExtension
	if logDir != "" {
		consoleLogPrefix = logDir + "/" + consoleLogPrefix
	}
	if logFile, err := os.Create(consoleLogPrefix); err != nil {
		log.Fatal("Unable to create logFile: ", err)
	} else {
		cLog.LogFileName = logFile
        cLog.AllowOnConsole = true
	}
}
