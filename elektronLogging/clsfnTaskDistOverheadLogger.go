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
	cLog.CreateLogFile(prefix)
	return cLog
}

func (cLog ClsfnTaskDistrOverheadLogger) Log(logType int, level log.Level, logData log.Fields, message string) {
	if cLog.Type == logType {

		logger.SetLevel(level)

		if cLog.AllowOnConsole {
			logger.SetOutput(os.Stdout)
			logger.WithFields(logData).Println(message)
		}

		logger.SetOutput(cLog.LogFile)
		logger.WithFields(logData).Println(message)
	}
	if cLog.next != nil {
		cLog.next.Log(logType, level, logData, message)
	}
}

func (cLog *ClsfnTaskDistrOverheadLogger) CreateLogFile(prefix string) {

	filename := strings.Join([]string{prefix, config.TaskDistrConfig.FilenameExtension}, "")
	dirName := logDir.getDirName()
	if dirName != "" {
		if logFile, err := os.Create(filepath.Join(dirName, filename)); err != nil {
			log.Fatal("Unable to create logFile: ", err)
		} else {
			cLog.LogFile = logFile
			cLog.AllowOnConsole = config.TaskDistrConfig.AllowOnConsole
		}
	}
}
