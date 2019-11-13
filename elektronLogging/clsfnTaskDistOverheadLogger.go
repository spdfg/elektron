package elektronLogging

import (
	"os"
	log "github.com/sirupsen/logrus"
)

type ClsfnTaskDistOverheadLogger struct {
	LoggerImpl
}

func NewClsfnTaskDistOverheadLogger(logType int, prefix string) *ClsfnTaskDistOverheadLogger {
	cLog := new(ClsfnTaskDistOverheadLogger)
	cLog.Type = logType
	cLog.SetLogFile(prefix)
	return cLog
}

func (cLog *ClsfnTaskDistOverheadLogger) Log(logType int, level log.Level, logData log.Fields,message string) {
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

	tskDistLogPrefix := prefix + config.TaskDistConfig.FilenameExtension
	if logDir != "" {
		tskDistLogPrefix = logDir + "/" + tskDistLogPrefix
	}
	if logFile, err := os.Create(tskDistLogPrefix); err != nil {
		log.Fatal("Unable to create logFile: ", err)
	} else {
		cLog.LogFileName = logFile
        cLog.AllowOnConsole = config.TaskDistConfig.AllowOnConsole
	}
}
