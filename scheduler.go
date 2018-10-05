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
	"gitlab.com/spdf/elektron/powerCap"
	"gitlab.com/spdf/elektron/schedulers"
)

var master = flag.String("master", "", "Location of leading Mesos master -- <mesos-master>:<port>")
var tasksFile = flag.String("workload", "", "JSON file containing task definitions")
var wattsAsAResource = flag.Bool("wattsAsAResource", false, "Enable Watts as a Resource")
var pcpConfigFile = flag.String("pcpConfigFile", "config", "PCP config file name (if file not "+
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
	flag.StringVar(pcpConfigFile, "pcpCF", "config", "PCP config file name (if not present in"+
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

	// Checking to see if we need to just list the pluggable scheduling policies
	if *listSchedPolicies {
		listAllSchedulingPolicies()
		os.Exit(1)
	}

	// Creating logger and attaching different logging platforms.
	startTime := time.Now()
	formattedStartTime := startTime.Format("20060102150405")
	// Checking if prefix contains any special characters
	if strings.Contains(*pcplogPrefix, "/") {
		log.Fatal("log file prefix should not contain '/'.")
	}
	logPrefix := *pcplogPrefix + "_" + formattedStartTime
	logger := elekLogDef.BuildLogger(startTime, logPrefix)
	// Logging channels.
	logMType := make(chan elekLogDef.LogMessageType)
	logMsg := make(chan string)
	go logger.Listen(logMType, logMsg)

	// First we need to build the scheduler using scheduler options.
	var schedOptions []schedulers.SchedulerOptions = make([]schedulers.SchedulerOptions, 0, 10)

	// OPTIONAL PARAMETERS
	// Scheduling Policy Name
	// If non-default scheduling policy given, checking if name exists.
	if *schedPolicyName != "first-fit" {
		if _, ok := schedulers.SchedPolicies[*schedPolicyName]; !ok {
			// invalid scheduling policy
			log.Println("Invalid scheduling policy given. The possible scheduling policies are:")
			listAllSchedulingPolicies()
			os.Exit(1)
		}
	}

	// CHANNELS AND FLAGS.
	shutdown := make(chan struct{})
	done := make(chan struct{})
	pcpLog := make(chan struct{})
	recordPCP := false

	// Logging channels.
	// These channels are used by the framework to log messages.
	// The channels are used to send the type of log message and the message string.
	schedOptions = append(schedOptions, schedulers.WithLoggingChannels(logMType, logMsg))

	// Shutdown indicator channels.
	// These channels are used to notify,
	// 1. scheduling is complete.
	// 2. all scheduled tasks have completed execution and framework can shutdown.
	schedOptions = append(schedOptions, schedulers.WithShutdown(shutdown))
	schedOptions = append(schedOptions, schedulers.WithDone(done))

	// If here, then valid scheduling policy name provided.
	schedOptions = append(schedOptions, schedulers.WithSchedPolicy(*schedPolicyName))

	// Scheduling Policy Switching.
	if *enableSchedPolicySwitch {
		// Scheduling policy config file required.
		if spcf := *schedPolConfigFile; spcf == "" {
			logger.WriteLog(elekLogDef.ERROR, "No file containing characteristics for"+
				" scheduling policies")
			os.Exit(1)
		} else {
			// Initializing the characteristics of the scheduling policies.
			schedulers.InitSchedPolicyCharacteristics(spcf)
			schedOptions = append(schedOptions, schedulers.WithSchedPolSwitchEnabled(*enableSchedPolicySwitch, *schedPolSwitchCriteria))
			// Fix First Scheduling Policy.
			schedOptions = append(schedOptions, schedulers.WithNameOfFirstSchedPolToFix(*fixFirstSchedPol))
			// Fix Scheduling Window.
			schedOptions = append(schedOptions, schedulers.WithFixedSchedulingWindow(*fixSchedWindow, *schedWindowSize))
		}
	}

	// Watts as a Resource (WaaR) and ClassMapWatts (CMW).
	// If WaaR and CMW is enabled then for each task the class_to_watts mapping is used to
	//      fit tasks into offers.
	// If CMW is disabled, then the Median of Medians Max Peak Power Usage value is used
	//	as the watts value for each task.
	if *wattsAsAResource {
		logger.WriteLog(elekLogDef.GENERAL, "WaaR enabled...")
		schedOptions = append(schedOptions, schedulers.WithWattsAsAResource(*wattsAsAResource))
		schedOptions = append(schedOptions, schedulers.WithClassMapWatts(*classMapWatts))
	}
	// REQUIRED PARAMETERS.
	// PCP logging, Power capping and High and Low thresholds.
	schedOptions = append(schedOptions, schedulers.WithRecordPCP(&recordPCP))
	schedOptions = append(schedOptions, schedulers.WithPCPLog(pcpLog))
	var noPowercap bool
	var extrema bool
	var progExtrema bool
	var powercapValues map[string]struct{} = map[string]struct{}{
		"":             {},
		"extrema":      {},
		"prog-extrema": {},
	}
	if _, ok := powercapValues[*powerCapPolicy]; !ok {
		logger.WriteLog(elekLogDef.ERROR, "Incorrect power-capping algorithm specified.")
		os.Exit(1)
	} else {
		// Indicating which power capping algorithm to use, if any.
		// The pcp-logging with/without power capping will be run after the
		// scheduler has been configured.
		if *powerCapPolicy == "" {
			noPowercap = true
		} else {
			if *powerCapPolicy == "extrema" {
				extrema = true
			} else if *powerCapPolicy == "prog-extrema" {
				progExtrema = true
			}
			// High and Low thresholds are currently only needed for extrema and
			// progressive extrema.
			if extrema || progExtrema {
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
	}

	// Tasks
	// If httpServer is disabled, then path of file containing workload needs to be provided.
	if *tasksFile == "" {
		logger.WriteLog(elekLogDef.ERROR, "No file containing tasks specification"+
			" provided.")
		os.Exit(1)
	}
	tasks, err := def.TasksFromJSON(*tasksFile)
	if err != nil || len(tasks) == 0 {
		logger.WriteLog(elekLogDef.ERROR, "Invalid tasks specification file "+
			"provided.")
		os.Exit(1)
	}
	schedOptions = append(schedOptions, schedulers.WithTasks(tasks))

	// Scheduler.
	scheduler := schedulers.SchedFactory(schedOptions...)

	// Scheduler driver.
	driver, err := sched.NewMesosSchedulerDriver(sched.DriverConfig{
		Master: *master,
		Framework: &mesos.FrameworkInfo{
			Name: proto.String("Elektron"),
			User: proto.String(""),
		},
		Scheduler: scheduler,
	})
	if err != nil {
		logger.WriteLog(elekLogDef.ERROR, fmt.Sprintf("Unable to create scheduler driver:"+
			" %s", err))
		os.Exit(1)
	}

	// Starting PCP logging.
	if noPowercap {
		go pcp.Start(pcpLog, &recordPCP, logMType, logMsg, *pcpConfigFile, scheduler)
	} else if extrema {
		go powerCap.StartPCPLogAndExtremaDynamicCap(pcpLog, &recordPCP, *hiThreshold,
			*loThreshold, logMType, logMsg, *pcpConfigFile, scheduler)
	} else if progExtrema {
		go powerCap.StartPCPLogAndProgressiveExtremaCap(pcpLog, &recordPCP, *hiThreshold,
			*loThreshold, logMType, logMsg, *pcpConfigFile, scheduler)
	}

	// Take a second between starting PCP log and continuing.
	time.Sleep(1 * time.Second)

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

		log.Println("Received SIGINT... stopping")
		close(done)
	}()

	go func() {

		// Signals we have scheduled every task we have
		select {
		case <-shutdown:
			//case <-time.After(shutdownTimeout):
		}

		// All tasks have finished
		select {
		case <-done:
			close(pcpLog)
			time.Sleep(5 * time.Second) //Wait for PCP to log a few more seconds
			// Closing logging channels.
			close(logMType)
			close(logMsg)
			//case <-time.After(shutdownTimeout):
		}

		// Done shutting down
		driver.Stop(false)

	}()

	// Starting the scheduler driver.
	if status, err := driver.Run(); err != nil {
		log.Printf("Framework stopped with status %s and error: %s\n", status.String(), err.Error())
	}
	log.Println("Exiting...")
}
