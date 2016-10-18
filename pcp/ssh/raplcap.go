package ssh

import (
	"golang.org/x/crypto/ssh"
	"fmt"
)

func main() {
	sshConfig := &ssh.ClientConfig{
		User: "rapl",
		Auth: []ssh.AuthMethod{
			ssh.Password("pankajlikesdanceswithwolves#!@#"),
		},
	}

	connection, err := ssh.Dial("tcp", "host:port", sshConfig)
	if err != nil {
		return nil, fmt.Errorf("Failed to dial: %s", err)
	}

	session, err := connection.NewSession()
	if err != nil {
		return nil, fmt.Errorf("Failed to create session: %s", err)
	}

	err = session.Run("sudo /misc/shared_data/rdelval1/RAPL_PKG_Throttle.py 100")
}