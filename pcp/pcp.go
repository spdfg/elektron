package pcp

import (
	elecLogDef "bitbucket.org/sunybingcloud/elektron/logging/def"
	"bitbucket.org/sunybingcloud/elektron/schedulers"
	"bufio"
	"fmt"
	"github.com/mesos/mesos-go/api/v0/scheduler"
	"log"
	"os/exec"
	"syscall"
	"time"
	"github.com/montanaflynn/stats"
	"github.com/mesos/mesos-go/api/v0/mesosproto"
	"path/filepath"
)

func Start(quit chan struct{}, logging *bool, logMType chan elecLogDef.LogMessageType, logMsg chan string, s scheduler.Scheduler) {
	baseSchedRef := s.(*schedulers.BaseScheduler)
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

		logMType <- elecLogDef.DEG_COL
		logMsg <- fmt.Sprintf("CPU Variance, CPU Task Share Variance, Memory Variance, Memory Task Share Variance")

		// Throw away first set of results
		scanner.Scan()

		seconds := 0

		for scanner.Scan() {
			text := scanner.Text()

			if *logging {
				logMType <- elecLogDef.PCP
				logMsg <- text
			}

			seconds++

			memUtils := memUtilPerNode(text)
			memTaskShares := make([]float64, len(memUtils))

			cpuUtils := cpuUtilPerNode(text)
			cpuTaskShares := make([]float64, len(cpuUtils))

			for i := 0; i < 8; i++ {
				host := fmt.Sprintf("stratos-00%d.cs.binghamton.edu", i+1)
				if slaveID, ok := baseSchedRef.HostNameToSlaveID[host]; ok {
					tasksRunning := len(baseSchedRef.Running[slaveID])
					if tasksRunning > 0 {
						cpuTaskShares[i] = cpuUtils[i] / float64(tasksRunning)
						memTaskShares[i] = memUtils[i] / float64(tasksRunning)
					}
				}
			}

			cpuVariance, _ := stats.Variance(cpuUtils)
			cpuTaskSharesVariance, _ := stats.Variance(cpuTaskShares)
			memVariance, _ := stats.Variance(memUtils)
			memTaskSharesVariance, _ := stats.Variance(memTaskShares)

			logMType <- elecLogDef.DEG_COL
			logMsg <- fmt.Sprintf("%f, %f, %f, %f", cpuVariance, cpuTaskSharesVariance, memVariance, memTaskSharesVariance)
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
