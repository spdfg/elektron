package elektronLogging

import (
	log "github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"strings"
)

type ClsfnTaskDistrOverheadLogger struct {
	LoggerImpl
}

func NewClsfnTaskDistrOverheadLogger(logType int, prefix string) *ClsfnTaskDistrOverheadLogger {
	cLog := &ClsfnTaskDistrOverheadLogger{}
	cLog.Type = logType
	cLog.SetLogFile(prefix)
	return cLog
}

func (cLog *ClsfnTaskDistrOverheadLogger) Log(logType int, level log.Level, logData log.Fields, message string) {
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

func (cLog *ClsfnTaskDistrOverheadLogger) SetLogFile(prefix string) {

	tskDistLogPrefix := strings.Join([]string{prefix, config.TaskDistConfig.FilenameExtension}, "")
	dirName := logDir.getDirName()
	if dirName != "" {
		tskDistLogPrefix = filepath.Join(dirName, tskDistLogPrefix)
	}
	if logFile, err := os.Create(tskDistLogPrefix); err != nil {
		log.Fatal("Unable to create logFile: ", err)
	} else {
		cLog.LogFileName = logFile
		cLog.AllowOnConsole = config.TaskDistConfig.AllowOnConsole
	}
}
