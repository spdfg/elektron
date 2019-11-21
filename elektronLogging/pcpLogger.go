package elektronLogging

import (
	log "github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"strings"
)

type PcpLogger struct {
	LoggerImpl
}

func NewPcpLogger(logType int, prefix string) *PcpLogger {
	pLog := &PcpLogger{}
	pLog.Type = logType
	pLog.SetLogFile(prefix)
	return pLog
}

func (pLog PcpLogger) Log(logType int, level log.Level, logData log.Fields, message string) {
	if pLog.Type == logType {

		logger.SetLevel(level)

		if pLog.AllowOnConsole {
			logger.SetOutput(os.Stdout)
			logger.WithFields(logData).Println(message)
		}

		logger.SetOutput(pLog.LogFileName)
		logger.WithFields(logData).Println(message)
	}
	if pLog.next != nil {
		pLog.next.Log(logType, level, logData, message)
	}
}

func (pLog *PcpLogger) SetLogFile(prefix string) {

	filename := strings.Join([]string{prefix, config.PCPConfig.FilenameExtension}, "")
	dirName := logDir.getDirName()
	if dirName != "" {
		if logFile, err := os.Create(filepath.Join(dirName, filename)); err != nil {
			log.Fatal("Unable to create logFile: ", err)
		} else {
			pLog.LogFileName = logFile
			pLog.AllowOnConsole = config.PCPConfig.AllowOnConsole
		}
	}
}
