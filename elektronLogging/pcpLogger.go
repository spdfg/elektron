package elektronLogging

import (
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

type PCPLogger struct {
	LoggerImpl
}

func NewPCPLogger(logType int, prefix string) *PCPLogger {
	pLog := &PCPLogger{}
	pLog.Type = logType
	pLog.CreateLogFile(prefix)
	return pLog
}

func (pLog PCPLogger) Log(logType int, level log.Level, logData log.Fields, message string) {
	if pLog.Type == logType {

		logger.SetLevel(level)

		if pLog.AllowOnConsole {
			logger.SetOutput(os.Stdout)
			logger.WithFields(logData).Println(message)
		}

		logger.SetOutput(pLog.LogFile)
		logger.WithFields(logData).Println(message)
	}
	if pLog.next != nil {
		pLog.next.Log(logType, level, logData, message)
	}
}

func (pLog *PCPLogger) CreateLogFile(prefix string) {

	filename := strings.Join([]string{prefix, config.PCPConfig.FilenameExtension}, "")
	dirName := logDir.getDirName()
	if dirName != "" {
		if logFile, err := os.Create(filepath.Join(dirName, filename)); err != nil {
			log.Fatal("Unable to create logFile: ", err)
		} else {
			pLog.LogFile = logFile
			pLog.AllowOnConsole = config.PCPConfig.AllowOnConsole
		}
	}
}
