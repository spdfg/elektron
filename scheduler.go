// Copyright (C) 2018 spdfg
//
// This file is part of Elektron.
//
// Elektron is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Elektron is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Elektron.  If not, see <http://www.gnu.org/licenses/>.
//

package main // import github.com/spdfg/elektron

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	mesos "github.com/mesos/mesos-go/api/v0/mesosproto"
	sched "github.com/mesos/mesos-go/api/v0/scheduler"
	log "github.com/sirupsen/logrus"
	"github.com/spdfg/elektron/def"
	elekLog "github.com/spdfg/elektron/logging"
	elekLogTypes "github.com/spdfg/elektron/logging/types"
	"github.com/spdfg/elektron/pcp"
	"github.com/spdfg/elektron/powerCap"
	"github.com/spdfg/elektron/schedulers"
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
var logConfigFilename = flag.String("logConfigFilename", "logConfig.yaml", "Log Configuration file name")

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
	flag.StringVar(logConfigFilename, "lgConfigName", "logConfig.yaml", "Log Configuration file name (shorthand).")
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
			log.Fatal("Scheduling policy characteristics file not provided.")
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
		log.Println("WaaR enabled...")
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
		log.Fatal("Incorrect power-capping algorithm specified.")
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
					log.Fatal("High threshold is of a lower value than low " +
						"threshold.")
				}
			}
		}
	}

	// Tasks
	// If httpServer is disabled, then path of file containing workload needs to be provided.
	if *tasksFile == "" {
		log.Fatal("Tasks specifications file not provided.")
	}
	tasks, err := def.TasksFromJSON(*tasksFile)
	if err != nil || len(tasks) == 0 {
		log.Fatal("Invalid tasks specification file provided.")
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
		log.Fatal(fmt.Sprintf("Unable to create scheduler driver: %s", err))
	}

	// Checking if prefix contains any special characters.
	if strings.Contains(*pcplogPrefix, "/") {
		log.Fatal("log file prefix should not contain '/'.")
	}
	// Build Logger for elektron.
	elekLog.BuildLogger(*pcplogPrefix, *logConfigFilename)

	// Starting PCP logging.
	if noPowercap {
		go pcp.Start(pcpLog, &recordPCP, *pcpConfigFile)
	} else if extrema {
		go powerCap.StartPCPLogAndExtremaDynamicCap(pcpLog, &recordPCP, *hiThreshold,
			*loThreshold, *pcpConfigFile)
	} else if progExtrema {
		go powerCap.StartPCPLogAndProgressiveExtremaCap(pcpLog, &recordPCP, *hiThreshold,
			*loThreshold, *pcpConfigFile)
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
			//case <-time.After(shutdownTimeout):
		}

		// Done shutting down
		driver.Stop(false)

	}()

	// Starting the scheduler driver.
	if status, err := driver.Run(); err != nil {
		elekLog.ElektronLogger.WithFields(log.Fields{"status": status.String(), "error": err.Error()}).Log(elekLogTypes.CONSOLE,
			log.ErrorLevel, "Framework stopped ")
	}
	elekLog.ElektronLogger.Log(elekLogTypes.CONSOLE, log.InfoLevel, "Exiting...")
}
