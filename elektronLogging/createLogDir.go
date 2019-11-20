package elektronLogging

import (
	logrus "github.com/sirupsen/logrus"
	"os"
	"strconv"
	"strings"
	"time"
)

var logDir string

func GetLogDir(startTime time.Time, prefix string) {
	if logDir == "" {
		logDir = createLogDir(prefix, startTime)
	}
}

func createLogDir(prefix string, startTime time.Time) string {

	// Creating directory to store all logs for this run
	logDirName := strings.Join([]string{"./", prefix, strconv.Itoa(startTime.Year())}, "")
	logDirName = strings.Join([]string{logDirName, startTime.Month().String(), strconv.Itoa(startTime.Day())}, "-")
	logDirName = strings.Join([]string{logDirName, strconv.Itoa(startTime.Hour())}, "_")
	logDirName = strings.Join([]string{logDirName, strconv.Itoa(startTime.Minute()), strconv.Itoa(startTime.Second())}, "-")

	if _, err := os.Stat(logDirName); os.IsNotExist(err) {
		os.Mkdir(logDirName, 0755)
	} else {
		logrus.Println("Unable to create log directory: ", err)
		logDirName = ""
	}
	return logDirName
}
