package elektronLogging

import (
	elekLog "github.com/sirupsen/logrus"
	"os"
	"strings"
)

type ConsoleLogger struct {
	LoggerImpl
}

func NewConsoleLogger(logType int, prefix string) *ConsoleLogger {
	cLog := &ConsoleLogger{}
	cLog.Type = logType
	cLog.SetLogFile(prefix)
	return cLog
}
func (cLog *ConsoleLogger) Log(logType int, level elekLog.Level, logData elekLog.Fields, message string) {
	if logType <= cLog.Type {

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

	consoleLogPrefix := strings.Join([]string{prefix, config.ConsoleConfig.FilenameExtension}, "")
	dirName := logDir.getDirName()
	if dirName != "" {
		consoleLogPrefix = strings.Join([]string{dirName, consoleLogPrefix}, "/")
	}
	if logFile, err := os.Create(consoleLogPrefix); err != nil {
		elekLog.Fatal("Unable to create logFile: ", err)
	} else {
		cLog.LogFileName = logFile
		cLog.AllowOnConsole = true
	}
}
