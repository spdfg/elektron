package schedulers

import (
	"bitbucket.org/sunybingcloud/elektron/def"
	mesos "github.com/mesos/mesos-go/mesosproto"
	sched "github.com/mesos/mesos-go/scheduler"
	"log"
)

// Implements mesos scheduler.
type ElectronScheduler interface {
	sched.Scheduler
	init(opts ...schedPolicyOption)
}

type base struct {
	ElectronScheduler
	tasksCreated     int
	tasksRunning     int
	tasks            []def.Task
	metrics          map[string]def.Metric
	running          map[string]map[string]bool
	wattsAsAResource bool
	classMapWatts    bool

	// First set of PCP values are garbage values, signal to logger to start recording when we're
	// about to schedule a new task.
	RecordPCP *bool

	// This channel is closed when the program receives an interrupt,
	// signalling that the program should shut down.
	Shutdown chan struct{}
	// This channel is closed after shutdown is closed, and only when all
	// outstanding tasks have been cleaned up.
	Done chan struct{}

	// Controls when to shutdown pcp logging.
	PCPLog chan struct{}

	schedTrace *log.Logger
}

func (s *base) init(opts ...schedPolicyOption) {
	for _, opt := range opts {
		// Applying options.
		if err := opt(s); err != nil {
			log.Fatal(err)
		}
	}
	s.running = make(map[string]map[string]bool)
}

func (s *base) OfferRescinded(_ sched.SchedulerDriver, offerID *mesos.OfferID) {
	log.Printf("Offer %s rescinded", offerID)
}
func (s *base) SlaveLost(_ sched.SchedulerDriver, slaveID *mesos.SlaveID) {
	log.Printf("Slave %s lost", slaveID)
}
func (s *base) ExecutorLost(_ sched.SchedulerDriver,
	executorID *mesos.ExecutorID,
	slaveID *mesos.SlaveID, status int) {
	log.Printf("Executor %s on slave %s was lost", executorID, slaveID)
}

func (s *base) Error(_ sched.SchedulerDriver, err string) {
	log.Printf("Receiving an error: %s", err)
}

func (s *base) FrameworkMessage(
	driver sched.SchedulerDriver,
	executorID *mesos.ExecutorID,
	slaveID *mesos.SlaveID,
	message string) {

	log.Println("Getting a framework message: ", message)
	log.Printf("Received a framework message from some unknown source: %s", *executorID.Value)
}

func (s *base) Registered(
	_ sched.SchedulerDriver,
	frameworkID *mesos.FrameworkID,
	masterInfo *mesos.MasterInfo) {
	log.Printf("Framework %s registered with master %s", frameworkID, masterInfo)
}

func (s *base) Reregistered(_ sched.SchedulerDriver, masterInfo *mesos.MasterInfo) {
	log.Printf("Framework re-registered with master %s", masterInfo)
}

func (s *base) Disconnected(sched.SchedulerDriver) {
	log.Println("Framework disconnected with master")
}
