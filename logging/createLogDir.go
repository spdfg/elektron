// Copyright (C) 2018 spdfg
//
// This file is part of Elektron.
//
// Elektron is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Elektron is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Elektron.  If not, see <http://www.gnu.org/licenses/>.
//

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
