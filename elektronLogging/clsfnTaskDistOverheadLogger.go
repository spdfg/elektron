package elektronLogging

import (
	elekLog "github.com/sirupsen/logrus"
	"os"
	"strings"
)

type ClsfnTaskDistOverheadLogger struct {
	LoggerImpl
}

func NewClsfnTaskDistOverheadLogger(logType int, prefix string) *ClsfnTaskDistOverheadLogger {
	cLog := &ClsfnTaskDistOverheadLogger{}
	cLog.Type = logType
	cLog.SetLogFile(prefix)
	return cLog
}

func (cLog *ClsfnTaskDistOverheadLogger) Log(logType int, level elekLog.Level, logData elekLog.Fields, message string) {
	if cLog.Type == logType {

		logger.SetLevel(level)

		if cLog.AllowOnConsole {
			logger.SetOutput(os.Stdout)
			logger.WithFields(logData).Println(message)
		}

		logger.SetOutput(cLog.LogFileName)
		logger.WithFields(logData).Println(message)
	}
	if cLog.next != nil {
		cLog.next.Log(logType, level, logData, message)
	}
}

func (cLog *ClsfnTaskDistOverheadLogger) SetLogFile(prefix string) {

	tskDistLogPrefix := strings.Join([]string{prefix, config.TaskDistConfig.FilenameExtension}, "")
	dirName := logDir.getDirName()
	if dirName != "" {
		tskDistLogPrefix = strings.Join([]string{dirName, tskDistLogPrefix}, "/")
	}
	if logFile, err := os.Create(tskDistLogPrefix); err != nil {
		elekLog.Fatal("Unable to create logFile: ", err)
	} else {
		cLog.LogFileName = logFile
		cLog.AllowOnConsole = config.TaskDistConfig.AllowOnConsole
	}
}
