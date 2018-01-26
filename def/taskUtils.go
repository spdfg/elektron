package def

import (
	"github.com/mash/gokmeans"
	"log"
	"sort"
	"errors"
	"fmt"
)

// Information about a cluster of tasks.
type TaskCluster struct {
	ClusterIndex int
	Tasks        []Task
	SizeScore    int // How many other clusters is this cluster bigger than
}

// Classification of Tasks using KMeans clustering using the watts consumption observations.
type TasksToClassify []Task

// Basic taskObservation calculator. This returns an array consisting of the MMPU requirements of a task.
func (tc TasksToClassify) taskObservationCalculator(task Task) []float64 {
	if task.ClassToWatts != nil {
		// Taking the aggregate.
		observations := []float64{}
		for _, watts := range task.ClassToWatts {
			observations = append(observations, watts)
		}
		return observations
	} else if task.Watts != 0.0 {
		return []float64{task.Watts}
	} else {
		log.Fatal("Unable to classify tasks. Missing Watts or ClassToWatts attribute in workload.")
		return []float64{0.0} // Won't reach here.
	}
}

func ClassifyTasks(tasks []Task, numberOfClusters int) []TaskCluster {
	tc := TasksToClassify(tasks)
	return tc.classify(numberOfClusters, tc.taskObservationCalculator)
}

func (tc TasksToClassify) classify(numberOfClusters int, taskObservation func(task Task) []float64) []TaskCluster {
	clusters := make(map[int][]Task)
	observations := getObservations(tc, taskObservation)
	// TODO: Make the max number of rounds configurable based on the size of the workload.
	// The max number of rounds (currently defaulted to 100) is the number of iterations performed to obtain
	// 	distinct clusters. When the data size becomes very large, we would need more iterations for clustering.
	if trained, centroids := gokmeans.Train(observations, numberOfClusters, 100); trained {
		for i := 0; i < len(observations); i++ {
			observation := observations[i]
			classIndex := gokmeans.Nearest(observation, centroids)
			if _, ok := clusters[classIndex]; ok {
				clusters[classIndex] = append(clusters[classIndex], tc[i])
			} else {
				clusters[classIndex] = []Task{tc[i]}
			}
		}
	}
	return labelAndOrder(clusters, numberOfClusters, taskObservation)
}

// Record observations.
func getObservations(tasks []Task, taskObservation func(task Task) []float64) []gokmeans.Node {
	observations := []gokmeans.Node{}
	for i := 0; i < len(tasks); i++ {
		observations = append(observations, taskObservation(tasks[i]))
	}
	return observations
}

// Size tasks based on the power consumption.
// TODO: Size the cluster in a better way other than just taking an aggregate of the watts resource requirement.
func clusterSize(tasks []Task, taskObservation func(task Task) []float64) float64 {
	size := 0.0
	for _, task := range tasks {
		for _, observation := range taskObservation(task) {
			size += observation
		}
	}
	return size
}

// Order clusters in increasing order of task heaviness.
func labelAndOrder(clusters map[int][]Task, numberOfClusters int, taskObservation func(task Task) []float64) []TaskCluster {
	// Determine the position of the cluster in the ordered list of clusters.
	sizedClusters := []TaskCluster{}

	// Initializing.
	for i := 0; i < numberOfClusters; i++ {
		sizedClusters = append(sizedClusters, TaskCluster{
			ClusterIndex: i,
			Tasks:        clusters[i],
			SizeScore:    0,
		})
	}

	for i := 0; i < numberOfClusters-1; i++ {
		// Sizing the current cluster.
		sizeI := clusterSize(clusters[i], taskObservation)

		// Comparing with the other clusters.
		for j := i + 1; j < numberOfClusters; j++ {
			sizeJ := clusterSize(clusters[j], taskObservation)
			if sizeI > sizeJ {
				sizedClusters[i].SizeScore++
			} else {
				sizedClusters[j].SizeScore++
			}
		}
	}

	// Sorting the clusters based on sizeScore.
	sort.SliceStable(sizedClusters, func(i, j int) bool {
		return sizedClusters[i].SizeScore <= sizedClusters[j].SizeScore
	})
	return sizedClusters
}

// Generic Task Sorter.
// Be able to sort an array of tasks based on any of the tasks' resources.
func SortTasks(ts []Task, sb sortBy) {
	sort.SliceStable(ts, func(i, j int) bool {
		return sb(&ts[i]) <= sb(&ts[j])
	})
}

// Map taskIDs to resource requirements.
type TaskResources struct {
	CPU   float64
	Ram   float64
	Watts float64
}

var taskResourceRequirement map[string]*TaskResources

// Record resource requirements for all the tasks.
func initTaskResourceRequirements(tasks []Task) {
	taskResourceRequirement = make(map[string]*TaskResources)
	baseTaskID := "electron-"
	for _, task := range tasks {
		for i := *task.Instances; i > 0; i-- {
			taskID := fmt.Sprintf("%s-%d", baseTaskID + task.Name, *task.Instances)
			taskResourceRequirement[taskID] = &TaskResources{
				CPU: task.CPU,
				Ram: task.RAM,
				Watts: task.Watts,
			}
		}
	}
}

// Retrieve the resource requirement of a task specified by the TaskID
func GetResourceRequirement(taskID string) (TaskResources, error) {
	if tr, ok := taskResourceRequirement[taskID]; ok {
		return *tr, nil
	} else {
		// Shouldn't be here.
		return TaskResources{}, errors.New("Invalid TaskID: " + taskID)
	}
}