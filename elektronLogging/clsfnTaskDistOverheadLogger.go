package elektronLogging

import (
	//"fmt"
	"os"
	logrus "github.com/sirupsen/logrus"
	data "gitlab.com/spdf/elektron/elektronLogging/data"
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

func (cLog *ClsfnTaskDistOverheadLogger) Log(logType int, level logrus.Level, logData data.LogData,message string) {
	if cLog.Type == logType {

		logFields := cloneFields(logData)
		
		log.SetLevel(level)
        
        if cLog.AllowOnConsole {
            log.SetOutput(os.Stdout)
		    log.WithFields(logFields).Println(message)
        }
		
		log.SetOutput(cLog.LogFileName)
		log.WithFields(logFields).Println(message)
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
		logrus.Fatal("Unable to create logFile: ", err)
	} else {
		cLog.LogFileName = logFile
        cLog.AllowOnConsole = config.TaskDistConfig.AllowOnConsole
	}
}
