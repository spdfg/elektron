package elektronLogging

import (
	"os"
	"strconv"
	"time"
	logrus "github.com/sirupsen/logrus"
)

var logDir string

func GetLogDir(startTime time.Time, prefix string) {
	if logDir == "" {
		logDir = createLogDir(prefix, startTime)
	}
}

func createLogDir(prefix string, startTime time.Time) string {
	// Creating directory to store all logs for this run
	logDirName := "./" + prefix + strconv.Itoa(startTime.Year())
	logDirName += "-"
	logDirName += startTime.Month().String()
	logDirName += "-"
	logDirName += strconv.Itoa(startTime.Day())
	logDirName += "_"
	logDirName += strconv.Itoa(startTime.Hour())
	logDirName += "-"
	logDirName += strconv.Itoa(startTime.Minute())
	logDirName += "-"
	logDirName += strconv.Itoa(startTime.Second())
	if _, err := os.Stat(logDirName); os.IsNotExist(err) {
		os.Mkdir(logDirName, 0755)
	} else {
		logrus.Println("Unable to create log directory: ", err)
		logDirName = ""
	}
	return logDirName
}
