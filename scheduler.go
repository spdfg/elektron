package main

import (
	"bitbucket.org/sunybingcloud/elektron/def"
	"bitbucket.org/sunybingcloud/elektron/pcp"
	"bitbucket.org/sunybingcloud/elektron/schedulers"
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

var master = flag.String("master", "<mesos-master>:5050", "Location of leading Mesos master")
var tasksFile = flag.String("workload", "", "JSON file containing task definitions")
var wattsAsAResource = flag.Bool("wattsAsAResource", false, "Enable Watts as a Resource. This allows the usage of the Watts attribute (if present) in the workload definition during offer matching.")
var pcplogPrefix = flag.String("logPrefix", "", "Prefix for PCP log file")
var hiThreshold = flag.Float64("hiThreshold", 0.0, "Upperbound for Cluster average historical power consumption, beyond which extrema/progressive-extrema would start power-capping")
var loThreshold = flag.Float64("loThreshold", 0.0, "Lowerbound for Cluster average historical power consumption, below which extrema/progressive-extrema would stop power-capping")
var classMapWatts = flag.Bool("classMapWatts", false, "Enable mapping of watts to powerClass of node")

// Short hand args
func init() {
	flag.StringVar(master, "m", "<mesos-master>:5050", "Location of leading Mesos master (shorthand)")
	flag.StringVar(tasksFile, "w", "", "JSON file containing task definitions (shorthand)")
	flag.BoolVar(wattsAsAResource, "waar", false, "Enable Watts as a Resource. " +
		"This allows the usage of the Watts attribute (if present) in the workload definition during offer matching. (shorthand)")
	flag.StringVar(pcplogPrefix, "p", "", "Prefix for PCP log file (shorthand)")
	flag.Float64Var(hiThreshold, "ht", 700.0, "Upperbound for Cluster average historical power consumption, " +
		"beyond which extrema/progressive-extrema would start power-capping (shorthand)")
	flag.Float64Var(loThreshold, "lt", 400.0, "Lowerbound for Cluster average historical power consumption, " +
		"below which extrema/progressive-extrema would stop power-capping (shorthand)")
	flag.BoolVar(classMapWatts, "cmw", false, "Enable mapping of watts to powerClass of node (shorthand)")
}

func main() {
	flag.Parse()

	if *tasksFile == "" {
		fmt.Println("No file containing tasks specifiction provided.")
		os.Exit(1)
	}

	if *hiThreshold < *loThreshold {
		fmt.Println("High threshold is of a lower value than low threshold.")
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
	startTime := time.Now().Format("20060102150405")
	logPrefix := *pcplogPrefix + "_" + startTime

	scheduler := schedulers.NewFirstFit(tasks, *wattsAsAResource, logPrefix, *classMapWatts)
	driver, err := sched.NewMesosSchedulerDriver(sched.DriverConfig{
		Master: *master,
		Framework: &mesos.FrameworkInfo{
			Name: proto.String("Elektron"),
			User: proto.String(""),
		},
		Scheduler: scheduler,
	})
	if err != nil {
		log.Printf("Unable to create scheduler driver: %s", err)
		return
	}

	go pcp.Start(scheduler.PCPLog, &scheduler.RecordPCP, logPrefix)
	//go pcp.StartPCPLogAndExtremaDynamicCap(scheduler.PCPLog, &scheduler.RecordPCP, logPrefix, *hiThreshold, *loThreshold)
	//go pcp.StartPCPLogAndProgressiveExtremaCap(scheduler.PCPLog, &scheduler.RecordPCP, logPrefix, *hiThreshold, *loThreshold)
	time.Sleep(1 * time.Second) // Take a second between starting PCP log and continuing

	// Attempt to handle SIGINT to not leave pmdumptext running
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
			//case <-time.After(shutdownTimeout):
		}

		// All tasks have finished
		select {
		case <-scheduler.Done:
			close(scheduler.PCPLog)
			time.Sleep(5 * time.Second) //Wait for PCP to log a few more seconds
			//case <-time.After(shutdownTimeout):
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
