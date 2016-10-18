package main

import (
	"bitbucket.org/bingcloud/electron/def"
	"bitbucket.org/bingcloud/electron/pcp"
	"bitbucket.org/bingcloud/electron/schedulers"
	"flag"
	"fmt"
	"github.com/golang/protobuf/proto"
	mesos "github.com/mesos/mesos-go/mesosproto"
	sched "github.com/mesos/mesos-go/scheduler"
	"log"
	"os"
	"os/signal"
	"time"
)

var master = flag.String("master", "xavier:5050", "Location of leading Mesos master")
var tasksFile = flag.String("workload", "", "JSON file containing task definitions")
var ignoreWatts = flag.Bool("ignoreWatts", false, "Ignore watts in offers")
var pcplogPrefix = flag.String("logPrefix", "", "Prefix for pcplog")
var hiThreshold = flag.Float64("hiThreshold", 0.0, "Upperbound for when we should start capping")
var loThreshold = flag.Float64("loThreshold", 0.0, "Lowerbound for when we should start uncapping")

// Short hand args
func init() {
	flag.StringVar(master, "m", "xavier:5050", "Location of leading Mesos master (shorthand)")
	flag.StringVar(tasksFile, "w", "", "JSON file containing task definitions (shorthand)")
	flag.BoolVar(ignoreWatts, "i", false, "Ignore watts in offers (shorthand)")
	flag.StringVar(pcplogPrefix, "p", "", "Prefix for pcplog (shorthand)")
	flag.Float64Var(hiThreshold, "ht", 700.0, "Upperbound for when we should start capping (shorthand)")
	flag.Float64Var(loThreshold, "lt", 400.0, "Lowerbound for when we should start uncapping (shorthand)")
}

func main() {
	flag.Parse()

	if *tasksFile == "" {
		fmt.Println("No file containing tasks specifiction provided.")
		os.Exit(1)
	}

	if *hiThreshold < *loThreshold {
		fmt.Println("High threshold is of a lower value than low threhold.")
		os.Exit(1)
	}

	tasks, err := def.TasksFromJSON(*tasksFile)
	if err != nil || len(tasks) == 0 {
		fmt.Println("Invalid tasks specification file provided")
		os.Exit(1)
	}

	log.Println("Scheduling the following tasks:")
	for _, task := range tasks {
		fmt.Println(task)
	}

	scheduler := schedulers.NewFirstFit(tasks, *ignoreWatts)
	driver, err := sched.NewMesosSchedulerDriver(sched.DriverConfig{
		Master: *master,
		Framework: &mesos.FrameworkInfo{
			Name: proto.String("Electron"),
			User: proto.String(""),
		},
		Scheduler: scheduler,
	})
	if err != nil {
		log.Printf("Unable to create scheduler driver: %s", err)
		return
	}

	//go pcp.Start(scheduler.PCPLog, &scheduler.RecordPCP, *pcplogPrefix)
	go pcp.StartLogAndDynamicCap(scheduler.PCPLog, &scheduler.RecordPCP, *pcplogPrefix, *hiThreshold, *loThreshold)
	time.Sleep(1 * time.Second)

	// Attempt to handle signint to not leave pmdumptext running
	// Catch interrupt
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, os.Kill)
		s := <-c
		if s != os.Interrupt {
			close(scheduler.PCPLog)
			return
		}

		log.Printf("Received SIGINT...stopping")
		close(scheduler.Done)
	}()

	go func() {

		// Signals we have scheduled every task we have
		select {
		case <-scheduler.Shutdown:
			//			case <-time.After(shutdownTimeout):
		}

		// All tasks have finished
		select {
		case <-scheduler.Done:
			close(scheduler.PCPLog)
			time.Sleep(5 * time.Second) //Wait for PCP to log a few more seconds
			//			case <-time.After(shutdownTimeout):
		}

		// Done shutting down
		driver.Stop(false)

	}()

	log.Printf("Starting...")
	if status, err := driver.Run(); err != nil {
		log.Printf("Framework stopped with status %s and error: %s\n", status.String(), err.Error())
	}
	log.Println("Exiting...")
}
