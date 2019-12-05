package logging

import (
	"os"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

type logDirectory struct {
	name string
}

func (logD *logDirectory) getDirName() string {
	return logD.name
}

func (logD *logDirectory) createLogDir(prefix string, startTime time.Time) {

	if logD.name == "" {
		// Creating directory to store all logs for this run. Directory name format : _2019-November-21_14-33-0.
		prefix = prefix + "_"
		logDirName := strings.Join([]string{"./", prefix, strconv.Itoa(startTime.Year())}, "")
		logDirName = strings.Join([]string{logDirName, startTime.Month().String(), strconv.Itoa(startTime.Day())}, "-")
		logDirName = strings.Join([]string{logDirName, strconv.Itoa(startTime.Hour())}, "_")
		logDirName = strings.Join([]string{logDirName, strconv.Itoa(startTime.Minute()), strconv.Itoa(startTime.Second())}, "-")

		if _, err := os.Stat(logDirName); os.IsNotExist(err) {
			os.Mkdir(logDirName, 0755)
		} else {
			log.Println("Unable to create log directory: ", err)
			logDirName = ""
		}

		logD.name = logDirName
	}
}
