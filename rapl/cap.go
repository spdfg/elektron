package rapl

import (
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
	"strconv"
)

func Cap(host, username string, percentage int) error {

	if percentage > 100 || percentage < 0 {
		return errors.New("Percentage is out of range")
	}

	sshConfig := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			// TODO: CHANGE and MAKE THIS USE SSH KEY BEFORE MAKING PUBLIC!!!!
			ssh.Password("pankajlikesdanceswithwolves#!@#"),
		},
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

	err = session.Run("sudo /misc/shared_data/rdelval1/RAPL_PKG_Throttle.py " + strconv.Itoa(percentage))
	if err != nil {
		return errors.Wrap(err, "Failed to run RAPL script")
	}

	return nil
}
