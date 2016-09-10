package main

import (
	"flag"
	"fmt"
	"github.com/golang/protobuf/proto"
	mesos "github.com/mesos/mesos-go/mesosproto"
	"github.com/mesos/mesos-go/mesosutil"
	sched "github.com/mesos/mesos-go/scheduler"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"time"
)

const (
	taskCPUs        = 0.1
	taskMem         = 32.0
	shutdownTimeout = time.Duration(30) * time.Second
)

const (
	dockerCommand = "echo Hello_World!"
)

var (
	defaultFilter = &mesos.Filters{RefuseSeconds: proto.Float64(1)}
)

// maxTasksForOffer computes how many tasks can be launched using a given offer
func maxTasksForOffer(offer *mesos.Offer) int {
	count := 0

	var cpus, mem, watts float64

	for _, resource := range offer.Resources {
		switch resource.GetName() {
		case "cpus":
			cpus += *resource.GetScalar().Value
		case "mem":
			mem += *resource.GetScalar().Value
		case "watts":
			watts += *resource.GetScalar().Value
			fmt.Println("Got watts!: ", *resource.GetScalar().Value)
		}
	}

	for cpus >= taskCPUs && mem >= taskMem {
		count++
		cpus -= taskCPUs
		mem -= taskMem
	}

	return count
}

// rendlerScheduler implements the Scheduler interface and stores
// the state needed for Rendler to function.
type electronScheduler struct {
	tasksCreated int
	tasksRunning int

	dockerExecutor *mesos.ExecutorInfo
	renderExecutor *mesos.ExecutorInfo

	// This channel is closed when the program receives an interrupt,
	// signalling that the program should shut down.
	shutdown chan struct{}
	// This channel is closed after shutdown is closed, and only when all
	// outstanding tasks have been cleaned up
	done chan struct{}
}

// New electron scheduler
func newElectronScheduler() *electronScheduler {
	rendlerArtifacts := executorURIs()

	s := &electronScheduler{

		dockerExecutor: &mesos.ExecutorInfo{
			ExecutorId: &mesos.ExecutorID{Value: proto.String("crawl-executor")},
			Command: &mesos.CommandInfo{
				Value: proto.String(dockerCommand),
				Uris:  rendlerArtifacts,
			},
			Name: proto.String("Crawler"),
		},

		shutdown: make(chan struct{}),
		done:     make(chan struct{}),
	}
	return s
}

func (s *electronScheduler) newTaskPrototype(offer *mesos.Offer) *mesos.TaskInfo {
	taskID := s.tasksCreated
	s.tasksCreated++
	return &mesos.TaskInfo{
		TaskId: &mesos.TaskID{
			Value: proto.String(fmt.Sprintf("RENDLER-%d", taskID)),
		},
		SlaveId: offer.SlaveId,
		Resources: []*mesos.Resource{
			mesosutil.NewScalarResource("cpus", taskCPUs),
			mesosutil.NewScalarResource("mem", taskMem),
		},
	}
}

func (s *electronScheduler) newCrawlTask(url string, offer *mesos.Offer) *mesos.TaskInfo {
	task := s.newTaskPrototype(offer)
	task.Name = proto.String("Electron_" + *task.TaskId.Value)
	task.Executor = s.dockerExecutor
	task.Data = []byte(url)
	return task
}

func (s *electronScheduler) Registered(
	_ sched.SchedulerDriver,
	frameworkID *mesos.FrameworkID,
	masterInfo *mesos.MasterInfo) {
	log.Printf("Framework %s registered with master %s", frameworkID, masterInfo)
}

func (s *electronScheduler) Reregistered(_ sched.SchedulerDriver, masterInfo *mesos.MasterInfo) {
	log.Printf("Framework re-registered with master %s", masterInfo)
}

func (s *electronScheduler) Disconnected(sched.SchedulerDriver) {
	log.Println("Framework disconnected with master")
}

func (s *electronScheduler) ResourceOffers(driver sched.SchedulerDriver, offers []*mesos.Offer) {
	log.Printf("Received %d resource offers", len(offers))
	for _, offer := range offers {
		select {
		case <-s.shutdown:
			log.Println("Shutting down: declining offer on [", offer.Hostname, "]")
			driver.DeclineOffer(offer.Id, defaultFilter)
			if s.tasksRunning == 0 {
				close(s.done)
			}
			continue
		default:
		}

		tasks := []*mesos.TaskInfo{}
		tasksToLaunch := maxTasksForOffer(offer)
		for tasksToLaunch > 0 {
			fmt.Println("There is enough resources to launch a task!")
		}

		if len(tasks) == 0 {
			driver.DeclineOffer(offer.Id, defaultFilter)
		} else {
			driver.LaunchTasks([]*mesos.OfferID{offer.Id}, tasks, defaultFilter)
		}
	}
}

func (s *electronScheduler) StatusUpdate(driver sched.SchedulerDriver, status *mesos.TaskStatus) {
	log.Printf("Received task status [%s] for task [%s]", NameFor(status.State), *status.TaskId.Value)

	if *status.State == mesos.TaskState_TASK_RUNNING {
		s.tasksRunning++
	} else if IsTerminal(status.State) {
		s.tasksRunning--
		if s.tasksRunning == 0 {
			select {
			case <-s.shutdown:
				close(s.done)
			default:
			}
		}
	}
}

func (s *electronScheduler) FrameworkMessage(
	driver sched.SchedulerDriver,
	executorID *mesos.ExecutorID,
	slaveID *mesos.SlaveID,
	message string) {

	log.Println("Getting a framework message: ", message)
	switch *executorID.Value {
	case *s.dockerExecutor.ExecutorId.Value:
		log.Print("Received framework message ", message)

	default:
		log.Printf("Received a framework message from some unknown source: %s", *executorID.Value)
	}
}

func (s *electronScheduler) OfferRescinded(_ sched.SchedulerDriver, offerID *mesos.OfferID) {
	log.Printf("Offer %s rescinded", offerID)
}
func (s *electronScheduler) SlaveLost(_ sched.SchedulerDriver, slaveID *mesos.SlaveID) {
	log.Printf("Slave %s lost", slaveID)
}
func (s *electronScheduler) ExecutorLost(_ sched.SchedulerDriver, executorID *mesos.ExecutorID, slaveID *mesos.SlaveID, status int) {
	log.Printf("Executor %s on slave %s was lost", executorID, slaveID)
}

func (s *electronScheduler) Error(_ sched.SchedulerDriver, err string) {
	log.Printf("Receiving an error: %s", err)
}

func executorURIs() []*mesos.CommandInfo_URI {
	basePath, err := filepath.Abs(filepath.Dir(os.Args[0]) + "/../..")
	if err != nil {
		log.Fatal("Failed to find the path to RENDLER")
	}
	baseURI := fmt.Sprintf("%s/", basePath)

	pathToURI := func(path string, extract bool) *mesos.CommandInfo_URI {
		return &mesos.CommandInfo_URI{
			Value:   &path,
			Extract: &extract,
		}
	}

	return []*mesos.CommandInfo_URI{
		pathToURI(baseURI+"render.js", false),
		pathToURI(baseURI+"python/crawl_executor.py", false),
		pathToURI(baseURI+"python/render_executor.py", false),
		pathToURI(baseURI+"python/results.py", false),
		pathToURI(baseURI+"python/task_state.py", false),
	}
}

func main() {
	master := flag.String("master", "127.0.1.1:5050", "Location of leading Mesos master")

	flag.Parse()

	scheduler := newElectronScheduler()
	driver, err := sched.NewMesosSchedulerDriver(sched.DriverConfig{
		Master: *master,
		Framework: &mesos.FrameworkInfo{
			Name: proto.String("RENDLER"),
			User: proto.String(""),
		},
		Scheduler: scheduler,
	})
	if err != nil {
		log.Printf("Unable to create scheduler driver: %s", err)
		return
	}

	// Catch interrupt
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, os.Kill)
		s := <-c
		if s != os.Interrupt {
			return
		}

		log.Println("Electron is shutting down")
		close(scheduler.shutdown)

		select {
		case <-scheduler.done:
		case <-time.After(shutdownTimeout):
		}

		// Done shutting down
		driver.Stop(false)
	}()

	if status, err := driver.Run(); err != nil {
		log.Printf("Framework stopped with status %s and error: %s\n", status.String(), err.Error())
	}
	log.Println("Exiting...")
}
