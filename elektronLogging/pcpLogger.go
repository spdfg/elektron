package elektronLogging

import (
	elekLog "github.com/sirupsen/logrus"
	"os"
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

func (pLog *PcpLogger) Log(logType int, level elekLog.Level, logData elekLog.Fields, message string) {
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

func (plog *PcpLogger) SetLogFile(prefix string) {

	pcpLogPrefix := strings.Join([]string{prefix, config.PCPConfig.FilenameExtension}, "")
	dirName := logDir.getDirName()
	if dirName != "" {
		pcpLogPrefix = strings.Join([]string{dirName, pcpLogPrefix}, "/")
	}
	if logFile, err := os.Create(pcpLogPrefix); err != nil {
		elekLog.Fatal("Unable to create logFile: ", err)
	} else {
		plog.LogFileName = logFile
		plog.AllowOnConsole = config.PCPConfig.AllowOnConsole
	}
}
