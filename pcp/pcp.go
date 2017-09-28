package pcp

import (
	"bufio"
	"log"
	"os"
	"os/exec"
	"syscall"
	"time"
)

func Start(quit chan struct{}, logging *bool, prefix string) {
	const pcpCommand string = "pmdumptext -m -l -f '' -t 1.0 -d , -c config"
	cmd := exec.Command("sh", "-c", pcpCommand)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	logFile, err := os.Create("./" + prefix + ".pcplog")
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Writing pcp logs to file: " + logFile.Name())

	defer logFile.Close()

	pipe, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	//cmd.Stdout = stdout

	scanner := bufio.NewScanner(pipe)

	go func(logging *bool) {
		// Get names of the columns.
		scanner.Scan()

		// Write to logfile.
		logFile.WriteString(scanner.Text() + "\n")

		// Throw away first set of results.
		scanner.Scan()

		seconds := 0
		for scanner.Scan() {

			if *logging {
				log.Println("Logging PCP...")
				logFile.WriteString(scanner.Text() + "\n")
			}

			seconds++
		}
	}(logging)

	log.Println("PCP logging started")

	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	pgid, err := syscall.Getpgid(cmd.Process.Pid)

	select {
	case <-quit:
		log.Println("Stopping PCP logging in 5 seconds")
		time.Sleep(5 * time.Second)

		// http://stackoverflow.com/questions/22470193/why-wont-go-kill-a-child-process-correctly
		// Kill process and all children processes.
		syscall.Kill(-pgid, 15)
		return
	}
}
