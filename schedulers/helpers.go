package schedulers

import (
	"bitbucket.org/sunybingcloud/elektron/constants"
	"bitbucket.org/sunybingcloud/elektron/def"
	"errors"
	"fmt"
	"log"
	"os"
)

func coLocated(tasks map[string]bool) {

	for task := range tasks {
		log.Println(task)
	}

	fmt.Println("---------------------")
}

// Get the powerClass of the given hostname
func hostToPowerClass(hostName string) string {
	for powerClass, hosts := range constants.PowerClasses {
		if _, ok := hosts[hostName]; ok {
			return powerClass
		}
	}
	return ""
}

// scheduler policy options to help initialize schedulers
type schedPolicyOption func(e ElectronScheduler) error

func WithTasks(ts []def.Task) schedPolicyOption {
	return func(s ElectronScheduler) error {
		if ts == nil {
			return errors.New("Task[] is empty.")
		} else {
			s.(*base).tasks = ts
			return nil
		}
	}
}

func WithWattsAsAResource(waar bool) schedPolicyOption {
	return func(s ElectronScheduler) error {
		s.(*base).wattsAsAResource = waar
		return nil
	}
}

func WithClassMapWatts(cmw bool) schedPolicyOption {
	return func(s ElectronScheduler) error {
		s.(*base).classMapWatts = cmw
		return nil
	}
}

func WithRecordPCP(recordPCP *bool) schedPolicyOption {
	return func(s ElectronScheduler) error {
		s.(*base).RecordPCP = recordPCP
		return nil
	}
}

func WithSchedTracePrefix(schedTracePrefix string) schedPolicyOption {
	return func(s ElectronScheduler) error {
		logFile, err := os.Create("./" + schedTracePrefix + "_schedTrace.log")
		if err != nil {
			return err
		} else {
			s.(*base).schedTrace = log.New(logFile, "", log.LstdFlags)
			return nil
		}
	}
}

func WithShutdown(shutdown chan struct{}) schedPolicyOption {
	return func(s ElectronScheduler) error {
		if shutdown == nil {
			return errors.New("Shutdown channel is nil.")
		} else {
			s.(*base).Shutdown = shutdown
			return nil
		}
	}
}

func WithDone(done chan struct{}) schedPolicyOption {
	return func(s ElectronScheduler) error {
		if done == nil {
			return errors.New("Done channel is nil.")
		} else {
			s.(*base).Done = done
			return nil
		}
	}
}

func WithPCPLog(pcpLog chan struct{}) schedPolicyOption {
	return func(s ElectronScheduler) error {
		if pcpLog == nil {
			return errors.New("PCPLog channel is nil.")
		} else {
			s.(*base).PCPLog = pcpLog
			return nil
		}
	}
}
