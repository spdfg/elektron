// Copyright (C) 2018 spdf
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
)

type ConsoleLogger struct {
	loggerObserverImpl
}

func (cl ConsoleLogger) Log(message string) {
	// We need to log to console only if the message is not empty
	if message != "" {
		log.Println(message)
		// Also logging the message to the console log file
		cl.logObserverSpecifics[conLogger].logFile.Println(message)
	}
}
