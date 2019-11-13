package elektronLogging

import (
	"os"
	log "github.com/sirupsen/logrus"
)

type SchedWindowLogger struct {
	LoggerImpl
}

func NewSchedWindowLogger(logType int, prefix string) *SchedWindowLogger {
	sLog := new(SchedWindowLogger)
	sLog.Type = logType
	sLog.SetLogFile(prefix)
	return sLog
}

func (sLog *SchedWindowLogger) Log(logType int, level log.Level, logData log.Fields, message string) {
	if sLog.Type == logType {

		logger.SetLevel(level)
		if sLog.AllowOnConsole {
            logger.SetOutput(os.Stdout)
		    logger.WithFields(logData).Println(message)
        }
    
		logger.SetOutput(sLog.LogFileName)
		logger.WithFields(logData).Println(message)
	}
	if sLog.next != nil {
		sLog.next.Log(logType, level, logData, message)
	}
}

func (sLog *SchedWindowLogger) SetLogFile(prefix string) {

	schedWindowLogPrefix := prefix + config.SchedWindowConfig.FilenameExtension
	if logDir != "" {
		schedWindowLogPrefix = logDir + "/" + schedWindowLogPrefix
	}
	if logFile, err := os.Create(schedWindowLogPrefix); err != nil {
		log.Fatal("Unable to create logFile: ", err)
	} else {
		sLog.LogFileName = logFile
        sLog.AllowOnConsole = config.SchedWindowConfig.AllowOnConsole
	}
}
