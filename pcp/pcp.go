package pcp

import (
	elecLogDef "bitbucket.org/sunybingcloud/elektron/logging/def"
	"bitbucket.org/sunybingcloud/elektron/schedulers"
	"bufio"
	"fmt"
	"github.com/mesos/mesos-go/api/v0/scheduler"
	"github.com/montanaflynn/stats"
	"log"
	"os/exec"
	"syscall"
	"time"
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
		logMsg <- "CPU Variance, CPU Task Share Variance, Memory Variance, Memory Task Share Variance"

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
					baseSchedRef.TasksRunningMutex.Lock()
					tasksRunning := len(baseSchedRef.Running[slaveID])
					baseSchedRef.TasksRunningMutex.Unlock()
					if tasksRunning > 0 {
						cpuTaskShares[i] = cpuUtils[i] / float64(tasksRunning)
						memTaskShares[i] = memUtils[i] / float64(tasksRunning)
					}
				}
			}

			// Variance in resource utilization shows how the current workload has been distributed.
			// However, if the number of tasks running are not equally distributed, utilization variance figures become
			// less relevant as they do not express the distribution of CPU intensive tasks.
			// We thus also calculate `task share variance`, which basically signifies how the workload is distributed
			// across each node per share.

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
