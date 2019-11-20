package elektronLogging

import (
	log "github.com/sirupsen/logrus"
	"os"
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
		logger.SetLevel(level)

		logger.SetOutput(os.Stdout)
		logger.WithFields(logData).Println(message)

		logger.SetOutput(cLog.LogFileName)
		logger.WithFields(logData).Println(message)
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
