package main

import (
	"bitbucket.org/sunybingcloud/elektron/def"
	"bitbucket.org/sunybingcloud/elektron/pcp"
	"bitbucket.org/sunybingcloud/elektron/schedulers"
        elecLogDef "bitbucket.org/sunybingcloud/elektron/logging/def"
	"flag"
	"fmt"
	"github.com/golang/protobuf/proto"
	mesos "github.com/mesos/mesos-go/mesosproto"
	sched "github.com/mesos/mesos-go/scheduler"
	"log"
	"os"
	"os/signal"
	"strings"
	"time"
)

var master = flag.String("master", "", "Location of leading Mesos master -- <mesos-master>:<port>")
var tasksFile = flag.String("workload", "", "JSON file containing task definitions")
var wattsAsAResource = flag.Bool("wattsAsAResource", false, "Enable Watts as a Resource")
var pcplogPrefix = flag.String("logPrefix", "", "Prefix for pcplog")
var hiThreshold = flag.Float64("hiThreshold", 0.0, "Upperbound for when we should start capping")
var loThreshold = flag.Float64("loThreshold", 0.0, "Lowerbound for when we should start uncapping")
var classMapWatts = flag.Bool("classMapWatts", false, "Enable mapping of watts to power class of node")
var schedPolicyName = flag.String("schedPolicy", "first-fit", "Name of the scheduling policy to be used.\n\tUse option -listSchedPolicies to get the names of available scheduling policies")
var listSchedPolicies = flag.Bool("listSchedPolicies", false, "List the names of the pluaggable scheduling policies.")
var enableSchedPolicySwitch = flag.Bool("switchSchedPolicy", false, "Enable switching of scheduling policies at runtime.")

// Short hand args
func init() {
	flag.StringVar(master, "m", "", "Location of leading Mesos master (shorthand)")
	flag.StringVar(tasksFile, "w", "", "JSON file containing task definitions (shorthand)")
	flag.BoolVar(wattsAsAResource, "waar", false, "Enable Watts as a Resource (shorthand)")
	flag.StringVar(pcplogPrefix, "p", "", "Prefix for pcplog (shorthand)")
	flag.Float64Var(hiThreshold, "ht", 700.0, "Upperbound for when we should start capping (shorthand)")
	flag.Float64Var(loThreshold, "lt", 400.0, "Lowerbound for when we should start uncapping (shorthand)")
	flag.BoolVar(classMapWatts, "cmw", false, "Enable mapping of watts to power class of node (shorthand)")
	flag.StringVar(schedPolicyName, "sp", "first-fit", "Name of the scheduling policy to be used.\n	Use option -listSchedPolicies to get the names of available scheduling policies (shorthand)")
	flag.BoolVar(listSchedPolicies, "lsp", false, "Names of the pluaggable scheduling policies. (shorthand)")
	flag.BoolVar(enableSchedPolicySwitch, "ssp", false, "Enable switching of scheduling policies at runtime.")
}

func listAllSchedulingPolicies() {
	fmt.Println("Scheduling Policies")
	fmt.Println("-------------------")
	for policyName, _ := range schedulers.SchedPolicies {
		fmt.Println(policyName)
	}
}

func main() {
	flag.Parse()

	// checking to see if we need to just list the pluggable scheduling policies
	if *listSchedPolicies {
		listAllSchedulingPolicies()
		os.Exit(1)
	}

	startTime := time.Now()
	formattedStartTime := startTime.Format("20060102150405")
	// Checking if prefix contains any special characters
	if strings.Contains(*pcplogPrefix, "/") {
		log.Fatal("log file prefix should not contain '/'.")
	}
	logPrefix := *pcplogPrefix + "_" + formattedStartTime

	// creating logger and attaching different logging platforms
	logger := elecLogDef.BuildLogger(startTime, logPrefix)
	// logging channels
	logMType := make(chan elecLogDef.LogMessageType)
	logMsg := make(chan string)
	go logger.Listen(logMType, logMsg)

	// If non-default scheduling policy given,
	// 	checking if scheduling policyName exists
	if *schedPolicyName != "first-fit" {
		if _, ok := schedulers.SchedPolicies[*schedPolicyName]; !ok {
			// invalid scheduling policy
			log.Println("Invalid scheduling policy given. The possible scheduling policies are:")
			listAllSchedulingPolicies()
			os.Exit(1)
		}
	}

	if *tasksFile == "" {
		//fmt.Println("No file containing tasks specifiction provided.")
		logger.WriteLog(elecLogDef.ERROR, "No file containing tasks specification provided")
		os.Exit(1)
	}

	if *hiThreshold < *loThreshold {
		//fmt.Println("High threshold is of a lower value than low threshold.")
		logger.WriteLog(elecLogDef.ERROR, "High threshold is of a lower value than low threshold")
		os.Exit(1)
	}

	tasks, err := def.TasksFromJSON(*tasksFile)
	if err != nil || len(tasks) == 0 {
		//fmt.Println("Invalid tasks specification file provided")
		logger.WriteLog(elecLogDef.ERROR, "Invalid tasks specification file provided")
		os.Exit(1)
	}

	//log.Println("Scheduling the following tasks:")
	logger.WriteLog(elecLogDef.GENERAL, "Scheduling the following tasks:")
	for _, task := range tasks {
		fmt.Println(task)
	}

	shutdown := make(chan struct{})
	done := make(chan struct{})
	pcpLog := make(chan struct{})
	recordPCP := false
	scheduler := schedulers.SchedFactory(
		schedulers.WithSchedPolicy(*schedPolicyName),
		schedulers.WithTasks(tasks),
		schedulers.WithWattsAsAResource(*wattsAsAResource),
		schedulers.WithClassMapWatts(*classMapWatts),
		schedulers.WithRecordPCP(&recordPCP),
		schedulers.WithShutdown(shutdown),
		schedulers.WithDone(done),
		schedulers.WithPCPLog(pcpLog),
		schedulers.WithLoggingChannels(logMType, logMsg),
		schedulers.WithSchedPolSwitchEnabled(*enableSchedPolicySwitch))
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

	go pcp.Start(pcpLog, &recordPCP, logMType, logMsg)
	//go pcp.StartPCPLogAndExtremaDynamicCap(pcpLog, &recordPCP, *hiThreshold, *loThreshold, logMType, logMsg)
	//go pcp.StartPCPLogAndProgressiveExtremaCap(pcpLog, &recordPCP, *hiThreshold, *loThreshold, logMType, logMsg)
	time.Sleep(1 * time.Second) // Take a second between starting PCP log and continuing

	// Attempt to handle SIGINT to not leave pmdumptext running.
	// Catch interrupt.
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, os.Kill)
		s := <-c
		if s != os.Interrupt {
			close(pcpLog)
			return
		}

		log.Printf("Received SIGINT...stopping")
		close(done)
	}()

	go func() {

		// Signals we have scheduled every task we have.
		select {
		case <-shutdown:
			//case <-time.After(shutdownTimeout):
		}

		// All tasks have finished.
		select {
		case <-done:
			close(pcpLog)
			time.Sleep(5 * time.Second) //Wait for PCP to log a few more seconds
			close(logMType)
			close(logMsg)
			//case <-time.After(shutdownTimeout):
		}

		// Done shutting down.
		driver.Stop(false)

	}()

	log.Printf("Starting...")
	if status, err := driver.Run(); err != nil {
		log.Printf("Framework stopped with status %s and error: %s\n", status.String(), err.Error())
	}
	log.Println("Exiting...")
}
