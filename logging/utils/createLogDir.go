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
	"log"
	"os"
	"strconv"
	"time"
)

var LogDir string

func GetLogDir(startTime time.Time, prefix string) string {
	if LogDir == "" {
		LogDir = createLogDir(prefix, startTime)
	}
	return LogDir
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
		log.Println("Unable to create log directory: ", err)
		logDirName = ""
	}
	return logDirName
}
