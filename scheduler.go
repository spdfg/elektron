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
	"time"
)

const (
	shutdownTimeout = time.Duration(30) * time.Second
)

const (
	dockerCommand = "echo Hello_World!"
)

var (
	defaultFilter = &mesos.Filters{RefuseSeconds: proto.Float64(1)}
)

type Task struct {
	cpu float64
	mem float64
	watts float64
	image string
}

// NameFor returns the string name for a TaskState.
func NameFor(state *mesos.TaskState) string {
	switch *state {
	case mesos.TaskState_TASK_STAGING:
		return "TASK_STAGING"
	case mesos.TaskState_TASK_STARTING:
		return "TASK_STARTING"
	case mesos.TaskState_TASK_RUNNING:
		return "TASK_RUNNING"
	case mesos.TaskState_TASK_FINISHED:
		return "TASK_FINISHED" // TERMINAL
	case mesos.TaskState_TASK_FAILED:
		return "TASK_FAILED" // TERMINAL
	case mesos.TaskState_TASK_KILLED:
		return "TASK_KILLED" // TERMINAL
	case mesos.TaskState_TASK_LOST:
		return "TASK_LOST" // TERMINAL
	default:
		return "UNKNOWN"
	}
}

// IsTerminal determines if a TaskState is a terminal state, i.e. if it singals
// that the task has stopped running.
func IsTerminal(state *mesos.TaskState) bool {
	switch *state {
	case mesos.TaskState_TASK_FINISHED,
		mesos.TaskState_TASK_FAILED,
		mesos.TaskState_TASK_KILLED,
		mesos.TaskState_TASK_LOST:
		return true
	default:
		return false
	}
}

// Decides if to take an offer or not
func offerDecision(offer *mesos.Offer) bool {

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

	var taskCPUs, taskMem, taskWatts float64

	// Insert calculation here
	taskWatts = 50
	taskMem = 4096
	taskCPUs = 3.0

	if cpus >= taskCPUs && mem >= taskMem && watts >= taskWatts {
		return true
	}

	return false
}

// rendlerScheduler implements the Scheduler interface and stores
// the state needed for Rendler to function.
type electronScheduler struct {
	tasksCreated int
	tasksRunning int
	taskQueue []Task //FIFO

	dockerExecutor *mesos.ExecutorInfo

	// This channel is closed when the program receives an interrupt,
	// signalling that the program should shut down.
	shutdown chan struct{}
	// This channel is closed after shutdown is closed, and only when all
	// outstanding tasks have been cleaned up
	done chan struct{}
}

// New electron scheduler
func newElectronScheduler() *electronScheduler {

	s := &electronScheduler{

		dockerExecutor: &mesos.ExecutorInfo{
			ExecutorId: &mesos.ExecutorID{Value: proto.String("docker-runner")},
			Command: &mesos.CommandInfo{
				Value: proto.String(dockerCommand),
			},
			Name: proto.String("Runner"),
		},

		shutdown: make(chan struct{}),
		done:     make(chan struct{}),
	}
	return s
}

func (s *electronScheduler) newTask(offer *mesos.Offer, taskCPUs, taskMem, taskWatts float64) *mesos.TaskInfo {
	taskID := s.tasksCreated
	s.tasksCreated++
	return &mesos.TaskInfo{
		TaskId: &mesos.TaskID{
			Value: proto.String(fmt.Sprintf("Electron-%d", taskID)),
		},
		SlaveId: offer.SlaveId,
		Resources: []*mesos.Resource{
			mesosutil.NewScalarResource("cpus", taskCPUs),
			mesosutil.NewScalarResource("mem", taskMem),
			mesosutil.NewScalarResource("watts", taskWatts),
		},
		Container: &mesos.ContainerInfo{
			Type: mesos.ContainerInfo_DOCKER.Enum(),
			Docker: &mesos.ContainerInfo_DockerInfo{
			Image: proto.String("gouravr/minife:v5"),
			},

		},
	}
}

func (s *electronScheduler) newDockerTask(offer *mesos.Offer, taskCPUs, taskMem, taskWatts float64) *mesos.TaskInfo {
	task := s.newTask(offer, taskCPUs, taskMem, taskWatts)
	task.Name = proto.String("Electron_" + *task.TaskId.Value)
	task.Command = &mesos.CommandInfo{
		Value: proto.String("cd src && mpirun -np 1 miniFE.x -nx 100 -ny 100 -nz 100"),
	}
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

		if offerDecision(offer) {
			tasks = append(tasks, s.newDockerTask(offer, 3.0, 4096, 50))
			driver.LaunchTasks([]*mesos.OfferID{offer.Id}, tasks, defaultFilter)
			time.Sleep(15 * time.Minute)
		} else {
			fmt.Println("There is enough resources to launch a task!")
			driver.DeclineOffer(offer.Id, defaultFilter)
			time.Sleep(15 * time.Minute)
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

func main() {
	master := flag.String("master", "xavier:5050", "Location of leading Mesos master")
	flag.Parse()

	scheduler := newElectronScheduler()
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

	log.Printf("Starting...")
	if status, err := driver.Run(); err != nil {
		log.Printf("Framework stopped with status %s and error: %s\n", status.String(), err.Error())
	}
	log.Println("Exiting...")
}
