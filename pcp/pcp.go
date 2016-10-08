package pcp

import (
	"bufio"
	"log"
	"os/exec"
	"time"
	"os"
	"syscall"
)

func Start(quit chan struct{}, logging *bool, prefix string) {
	const pcpCommand string = "pmdumptext -m -l -f '' -t 1.0 -d , -c config"
	cmd := exec.Command("sh", "-c", pcpCommand)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	startTime := time.Now().Format("20060102150405")


	logFile, err := os.Create("./"+prefix+startTime+".pcplog")
	if err != nil {
		log.Fatal(err)
	}

	defer logFile.Close()

	pipe, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	//cmd.Stdout = stdout

	scanner := bufio.NewScanner(pipe)

	go func(logging *bool) {
		// Get names of the columns
		scanner.Scan()

		// Write to logfile
		logFile.WriteString(scanner.Text() + "\n")

		/*
		headers := strings.Split(scanner.Text(), ",")

		for _, hostMetric := range headers {
			split := strings.Split(hostMetric, ":")
			fmt.Printf("Host %s: Metric: %s\n", split[0], split[1])
		}
		*/

		// Throw away first set of results
		scanner.Scan()

		seconds := 0
		for scanner.Scan() {


			if(*logging) {
				log.Println("Logging PCP...")
				logFile.WriteString(scanner.Text() + "\n")
			}

			/*
			fmt.Printf("Second: %d\n", seconds)
			for i, val := range strings.Split(scanner.Text(), ",") {
				fmt.Printf("host metric: %s val: %s\n", headers[i], val)
			}*/

			seconds++

			// fmt.Println("--------------------------------")
		}
	}(logging)

	log.Println("PCP logging started")


	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	pgid, err := syscall.Getpgid(cmd.Process.Pid)

	select{
	case <- quit:
		log.Println("Stopping PCP logging in 5 seconds")
		time.Sleep(5 * time.Second)

		// http://stackoverflow.com/questions/22470193/why-wont-go-kill-a-child-process-correctly
		// kill process and all children processes
		syscall.Kill(-pgid, 15)
		return
	}
}
