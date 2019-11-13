package elektronLogging

import (
	"os"
	log "github.com/sirupsen/logrus"
	//data "github.com/spdfg/elektron/elektronLogging/data"
)

type PcpLogger struct {
	LoggerImpl
}

func NewPcpLogger(logType int, prefix string) *PcpLogger {
	pLog := new(PcpLogger)
	pLog.Type = logType
	pLog.SetLogFile(prefix)
	return pLog
}

func (pLog *PcpLogger) Log(logType int, level log.Level, logData log.Fields, message string) {
	if pLog.Type == logType {

		//logFields := cloneFields(logData)
		
		log.SetLevel(level)
		
        if pLog.AllowOnConsole {
            log.SetOutput(os.Stdout)
		    log.WithFields(logData).Println(message)
        }

		log.SetOutput(pLog.LogFileName)
		log.WithFields(logData).Println(message)
	}
	if pLog.next != nil {
		pLog.next.Log(logType, level, logData, message)
	}
}

func (plog *PcpLogger) SetLogFile(prefix string) {

	pcpLogPrefix := prefix + config.PCPConfig.FilenameExtension
	if logDir != "" {
		pcpLogPrefix = logDir + "/" + pcpLogPrefix
	}
	if logFile, err := os.Create(pcpLogPrefix); err != nil {
		log.Fatal("Unable to create logFile: ", err)
	} else {
		plog.LogFileName = logFile
        plog.AllowOnConsole = config.PCPConfig.AllowOnConsole
	}
}
