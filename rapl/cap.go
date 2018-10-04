package rapl

import (
	"os"
	"strconv"

	"github.com/pkg/errors"
	elekEnv "gitlab.com/spdf/elektron/environment"
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
