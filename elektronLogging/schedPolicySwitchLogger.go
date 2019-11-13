package elektronLogging

import (
	//"fmt"
	"os"
	log "github.com/sirupsen/logrus"
	//data "github.com/spdfg/elektron/elektronLogging/data"
)

type SchedPolicySwitchLogger struct {
	LoggerImpl
}

func NewSchedPolicySwitchLogger(logType int, prefix string) *SchedPolicySwitchLogger {
	sLog := new(SchedPolicySwitchLogger)
	sLog.Type = logType
	sLog.SetLogFile(prefix)
	return sLog
}

func (sLog *SchedPolicySwitchLogger) Log(logType int, level log.Level, logData log.Fields, message string) {
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

func (sLog *SchedPolicySwitchLogger) SetLogFile(prefix string) {

	spsLogPrefix := prefix + config.SPSConfig.FilenameExtension
	if logDir != "" {
		spsLogPrefix = logDir + "/" + spsLogPrefix
	}
	if logFile, err := os.Create(spsLogPrefix); err != nil {
		log.Fatal("Unable to create logFile: ", err)
	} else {
		sLog.LogFileName = logFile
        sLog.AllowOnConsole = config.SPSConfig.AllowOnConsole
	}
}
