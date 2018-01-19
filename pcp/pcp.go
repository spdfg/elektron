package pcp

import (
	elecLogDef "bitbucket.org/sunybingcloud/elektron/logging/def"
	"bufio"
	"log"
	"os/exec"
	"syscall"
	"time"
)

func Start(quit chan struct{}, logging *bool, logMType chan elecLogDef.LogMessageType, logMsg chan string) {
	const pcpCommand string = "pmdumptext -m -l -f '' -t 1.0 -d , -c config"
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
		logMType <- elecLogDef.PCP
		logMsg <- scanner.Text()

		// Throw away first set of results.
		scanner.Scan()

		seconds := 0
		for scanner.Scan() {

			if *logging {
				logMType <- elecLogDef.PCP
				logMsg <- scanner.Text()
			}

			seconds++
		}
	}(logging)

	logMType <- elecLogDef.GENERAL
	logMsg <- "PCP logging started"

	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	pgid, err := syscall.Getpgid(cmd.Process.Pid)

	select {
	case <-quit:
		logMType <- elecLogDef.GENERAL
		logMsg <- "Stopping PCP logging in 5 seconds"
		time.Sleep(5 * time.Second)

		// http://stackoverflow.com/questions/22470193/why-wont-go-kill-a-child-process-correctly
		// Kill process and all children processes.
		syscall.Kill(-pgid, 15)
		return
	}
}
