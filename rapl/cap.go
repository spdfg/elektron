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

package rapl

import (
	"os"
	"strconv"

	"github.com/pkg/errors"
	elekEnv "github.com/spdfg/elektron/environment"
	"golang.org/x/crypto/ssh"
	"strings"
)

func Cap(host, username string, percentage float64) error {

	if percentage > 100 || percentage < 0 {
		return errors.New("Percentage is out of range")
	}

	sshConfig := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			// TODO: CHANGE and MAKE THIS USE SSH KEY!!!!
			ssh.Password(os.Getenv(elekEnv.RaplPassword)),
		},
		// TODO Do not allow accepting any host key.
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	connection, err := ssh.Dial("tcp", host+":22", sshConfig)
	if err != nil {
		return errors.Wrap(err, "Failed to dial")
	}

	session, err := connection.NewSession()
	defer session.Close()
	if err != nil {
		return errors.Wrap(err, "Failed to create session")
	}

	err = session.Run(strings.Join([]string{"sudo", os.Getenv(elekEnv.RaplThrottleScriptLocation),
		strconv.FormatFloat(percentage, 'f', 2, 64)}, " "))
	if err != nil {
		return errors.Wrap(err, "Failed to run RAPL script")
	}

	return nil
}
