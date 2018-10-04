package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	mesos "github.com/mesos/mesos-go/api/v0/mesosproto"
	sched "github.com/mesos/mesos-go/api/v0/scheduler"
	"gitlab.com/spdf/elektron/def"
	elekLogDef "gitlab.com/spdf/elektron/logging/def"
	"gitlab.com/spdf/elektron/pcp"
	"gitlab.com/spdf/elektron/schedulers"
	"gitlab.com/spdf/elektron/powerCap"
)

var master = flag.String("master", "", "Location of leading Mesos master -- <mesos-master>:<port>")
var tasksFile = flag.String("workload", "", "JSON file containing task definitions")
var wattsAsAResource = flag.Bool("wattsAsAResource", false, "Enable Watts as a Resource")
var pcpConfigFile = flag.String("pcpConfigFile", "config", "PCP config file name (if file not " +
	"present in the same directory, then provide path).")
var pcplogPrefix = flag.String("logPrefix", "", "Prefix for pcplog")
var powerCapPolicy = flag.String("powercap", "", "Power Capping policy. (default (''), extrema, prog-extrema).")
var hiThreshold = flag.Float64("hiThreshold", 0.0, "Upperbound for when we should start capping")
var loThreshold = flag.Float64("loThreshold", 0.0, "Lowerbound for when we should start uncapping")
var classMapWatts = flag.Bool("classMapWatts", false, "Enable mapping of watts to power class of node")
var schedPolicyName = flag.String("schedPolicy", "first-fit", "Name of the scheduling policy to be used.\n\tUse option -listSchedPolicies to get the names of available scheduling policies")
var listSchedPolicies = flag.Bool("listSchedPolicies", false, "List the names of the pluaggable scheduling policies.")
var enableSchedPolicySwitch = flag.Bool("switchSchedPolicy", false, "Enable switching of scheduling policies at runtime.")
var schedPolConfigFile = flag.String("schedPolConfig", "", "Config file that contains information for each scheduling policy.")
var fixFirstSchedPol = flag.String("fixFirstSchedPol", "", "Name of the scheduling policy to be deployed first, regardless of the distribution of tasks, provided switching is enabled.")
var fixSchedWindow = flag.Bool("fixSchedWindow", false, "Fix the size of the scheduling window that every deployed scheduling policy should schedule, provided switching is enabled.")
var schedWindowSize = flag.Int("schedWindowSize", 200, "Size of the scheduling window if fixSchedWindow is set.")
var schedPolSwitchCriteria = flag.String("schedPolSwitchCriteria", "taskDist", "Scheduling policy switching criteria.")

// Short hand args
func init() {
	flag.StringVar(master, "m", "", "Location of leading Mesos master (shorthand)")
	flag.StringVar(tasksFile, "w", "", "JSON file containing task definitions (shorthand)")
	flag.BoolVar(wattsAsAResource, "waar", false, "Enable Watts as a Resource (shorthand)")
	flag.StringVar(pcpConfigFile, "pcpCF", "config", "PCP config file name (if not present in" +
		" the same directory, then provide path) (shorthand).")
	flag.StringVar(pcplogPrefix, "p", "", "Prefix for pcplog (shorthand)")
        flag.StringVar(powerCapPolicy, "pc", "", "Power Capping policy. (default (''), extrema, prog-extrema) (shorthand).")
	flag.Float64Var(hiThreshold, "ht", 700.0, "Upperbound for when we should start capping (shorthand)")
	flag.Float64Var(loThreshold, "lt", 400.0, "Lowerbound for when we should start uncapping (shorthand)")
	flag.BoolVar(classMapWatts, "cmw", false, "Enable mapping of watts to power class of node (shorthand)")
	flag.StringVar(schedPolicyName, "sp", "first-fit", "Name of the scheduling policy to be used.\n	Use option -listSchedPolicies to get the names of available scheduling policies (shorthand)")
	flag.BoolVar(listSchedPolicies, "lsp", false, "Names of the pluaggable scheduling policies. (shorthand)")
	flag.BoolVar(enableSchedPolicySwitch, "ssp", false, "Enable switching of scheduling policies at runtime.")
	flag.StringVar(schedPolConfigFile, "spConfig", "", "Config file that contains information for each scheduling policy (shorthand).")
	flag.StringVar(fixFirstSchedPol, "fxFstSchedPol", "", "Name of the scheduling gpolicy to be deployed first, regardless of the distribution of tasks, provided switching is enabled (shorthand).")
	flag.BoolVar(fixSchedWindow, "fixSw", false, "Fix the size of the scheduling window that every deployed scheduling policy should schedule, provided switching is enabled (shorthand).")
	flag.IntVar(schedWindowSize, "swSize", 200, "Size of the scheduling window if fixSchedWindow is set (shorthand).")
	flag.StringVar(schedPolSwitchCriteria, "spsCriteria", "taskDist", "Scheduling policy switching criteria (shorthand).")
}

func listAllSchedulingPolicies() {
	fmt.Println("Scheduling Policies")
	fmt.Println("-------------------")
	for policyName := range schedulers.SchedPolicies {
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
	logger := elekLogDef.BuildLogger(startTime, logPrefix)
	// logging channels
	logMType := make(chan elekLogDef.LogMessageType)
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
		logger.WriteLog(elekLogDef.ERROR, "No file containing tasks specification provided")
		os.Exit(1)
	}

	if *hiThreshold < *loThreshold {
		//fmt.Println("High threshold is of a lower value than low threshold.")
		logger.WriteLog(elekLogDef.ERROR, "High threshold is of a lower value than low threshold")
		os.Exit(1)
	}

	tasks, err := def.TasksFromJSON(*tasksFile)
	if err != nil || len(tasks) == 0 {
		//fmt.Println("Invalid tasks specification file provided")
		logger.WriteLog(elekLogDef.ERROR, "Invalid tasks specification file provided")
		os.Exit(1)
	}

	//log.Println("Scheduling the following tasks:")
	logger.WriteLog(elekLogDef.GENERAL, "Scheduling the following tasks:")
	for _, task := range tasks {
		fmt.Println(task)
	}

	if *enableSchedPolicySwitch {
		if spcf := *schedPolConfigFile; spcf == "" {
			logger.WriteLog(elekLogDef.ERROR, "No file containing characteristics for scheduling policies")
		} else {
			// Initializing the characteristics of the scheduling policies.
			schedulers.InitSchedPolicyCharacteristics(spcf)
		}
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
		schedulers.WithSchedPolSwitchEnabled(*enableSchedPolicySwitch, *schedPolSwitchCriteria),
		schedulers.WithNameOfFirstSchedPolToFix(*fixFirstSchedPol),
		schedulers.WithFixedSchedulingWindow(*fixSchedWindow, *schedWindowSize))
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

	// Power Capping policy (if required).
	var noPowercap, extrema, progExtrema bool
	var powercapValues map[string]struct{} = map[string]struct{} {
		"": {},
		"extrema": {},
		"progExtrema": {},
	}

	if _, ok := powercapValues[*powerCapPolicy]; !ok {
		logger.WriteLog(elekLogDef.ERROR, "Incorrect power capping policy specified.")
		os.Exit(1)
	} else {
		// Indicating which power capping policy to use, if any.
		if *powerCapPolicy == "" {
			noPowercap = true
		} else {
			if *powerCapPolicy == "extrema" {
				extrema = true
			} else {
				progExtrema = true
			}
			// High and Low Thresholds.
			// These values are not used to configure the scheduler.
			// hiThreshold and loThreshold are passed to the powercappers.
			if *hiThreshold < *loThreshold {
				logger.WriteLog(elekLogDef.ERROR, "High threshold is of a"+
					" lower value than low threshold.")
				os.Exit(1)
			}
		}
	}

	// Checking if pcp config file exists.
	if _, err := os.Stat(*pcpConfigFile); os.IsNotExist(err) {
		logger.WriteLog(elekLogDef.ERROR, "PCP config file does not exist!")
		os.Exit(1)
	}

	if noPowercap {
		go pcp.Start(pcpLog, &recordPCP, logMType, logMsg, *pcpConfigFile, scheduler)
	} else if extrema {
		go powerCap.StartPCPLogAndExtremaDynamicCap(pcpLog, &recordPCP, *hiThreshold,
			*loThreshold, logMType, logMsg, *pcpConfigFile)
	} else if progExtrema {
		go powerCap.StartPCPLogAndProgressiveExtremaCap(pcpLog, &recordPCP, *hiThreshold,
			*loThreshold, logMType, logMsg, *pcpConfigFile)
	}

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

		log.Println("Received SIGINT...stopping")
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

	log.Println("Starting...")
	if status, err := driver.Run(); err != nil {
		log.Printf("Framework stopped with status %s and error: %s\n", status.String(), err.Error())
	}
	log.Println("Exiting...")
}
