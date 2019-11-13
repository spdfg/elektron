package elektronLogging

import (
	//"fmt"
	"os"
	log "github.com/sirupsen/logrus"
	//data "github.com/spdfg/elektron/elektronLogging/data"
)

type SchedTraceLogger struct {
	LoggerImpl
}

func NewSchedTraceLogger(logType int, prefix string) *SchedTraceLogger {
	sLog := new(SchedTraceLogger)
	sLog.Type = logType
	sLog.SetLogFile(prefix)
	return sLog
}

func (sLog *SchedTraceLogger) Log(logType int, level log.Level, logData log.Fields, message string) {
	if sLog.Type == logType {

		//logFields := cloneFields(logData)
		
		log.SetLevel(level)
        
        if sLog.AllowOnConsole {
            log.SetOutput(os.Stdout)
		    log.WithFields(logData).Println(message)
        }
		
		log.SetOutput(sLog.LogFileName)
		log.WithFields(logData).Println(message)
	}
	if sLog.next != nil {
		sLog.next.Log(logType, level, logData, message)
	}
}

func (sLog *SchedTraceLogger) SetLogFile(prefix string) {

	schedTraceLogPrefix := prefix + config.SchedTraceConfig.FilenameExtension
	if logDir != "" {
		schedTraceLogPrefix = logDir + "/" + schedTraceLogPrefix
	}
	if logFile, err := os.Create(schedTraceLogPrefix); err != nil {
		log.Fatal("Unable to create logFile: ", err)
	} else {
		sLog.LogFileName = logFile
        sLog.AllowOnConsole = config.SchedTraceConfig.AllowOnConsole
	}
}
