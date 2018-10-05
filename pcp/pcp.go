package pcp

import (
	"bufio"
	"log"
	"os/exec"
	"syscall"
	"time"

	elekLogDef "gitlab.com/spdf/elektron/logging/def"
)

func Start(quit chan struct{}, logging *bool, logMType chan elekLogDef.LogMessageType,
	logMsg chan string, pcpConfigFile string) {
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
		logMType <- elekLogDef.PCP
		logMsg <- scanner.Text()

		// Throw away first set of results
		scanner.Scan()

		seconds := 0

		for scanner.Scan() {
			text := scanner.Text()

			if *logging {
				logMType <- elekLogDef.PCP
				logMsg <- text
			}

			seconds++
		}
	}(logging)

	logMType <- elekLogDef.GENERAL
	logMsg <- "PCP logging started"

	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	pgid, err := syscall.Getpgid(cmd.Process.Pid)

	select {
	case <-quit:
		logMType <- elekLogDef.GENERAL
		logMsg <- "Stopping PCP logging in 5 seconds"
		time.Sleep(5 * time.Second)

		// http://stackoverflow.com/questions/22470193/why-wont-go-kill-a-child-process-correctly
		// Kill process and all children processes.
		syscall.Kill(-pgid, 15)
		return
	}
}
