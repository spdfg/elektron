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

package pcp

import (
	"bufio"
	"os/exec"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spdfg/elektron/elektronLogging"
	elekLogT "github.com/spdfg/elektron/elektronLogging/types"
)

func Start(quit chan struct{}, logging *bool, pcpConfigFile string) {
	var pcpCommand string = "pmdumptext -m -l -f '' -t 1.0 -d , -c " + pcpConfigFile
	cmd := exec.Command("sh", "-c", pcpCommand)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	pipe, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	//cmd.Stdout = stdout

	scanner := bufio.NewScanner(pipe)

	go func(logging *bool) {
		// Get names of the columns.
		scanner.Scan()

		// Write to logfile
		elektronLogging.ElektronLog.Log(elekLogT.PCP,
			log.InfoLevel,
			log.Fields{}, scanner.Text())

		// Throw away first set of results
		scanner.Scan()

		seconds := 0

		for scanner.Scan() {
			text := scanner.Text()

			if *logging {
				elektronLogging.ElektronLog.Log(elekLogT.PCP,
					log.InfoLevel,
					log.Fields{}, text)
			}

			seconds++
		}
	}(logging)

	elektronLogging.ElektronLog.Log(elekLogT.GENERAL,
		log.InfoLevel,
		log.Fields{}, "PCP logging started")

	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	pgid, err := syscall.Getpgid(cmd.Process.Pid)

	select {
	case <-quit:
		elektronLogging.ElektronLog.Log(elekLogT.GENERAL,
			log.InfoLevel,
			log.Fields{}, "Stopping PCP logging in 5 seconds")
		time.Sleep(5 * time.Second)

		// http://stackoverflow.com/questions/22470193/why-wont-go-kill-a-child-process-correctly
		// Kill process and all children processes.
		syscall.Kill(-pgid, 15)
		return
	}
}
