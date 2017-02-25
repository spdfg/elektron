package def

import (
	"github.com/mdesenfants/gokmeans"
	"sort"
)

// The watts consumption observations are taken into consideration.
func getObservations(tasks []Task) []gokmeans.Node {
	observations := []gokmeans.Node{}
	for i := 0; i < len(tasks); i++ {
		task := tasks[i]
		// If observations present for the power-classes, then using it
		if task.ClassToWatts != nil {
			observation := gokmeans.Node{}
			for _, watts := range task.ClassToWatts {
				observation = append(observation, watts)
			}
			observations = append(observations, observation)
		} else {
			// Using the watts attribute alone
			observations = append(observations, gokmeans.Node{task.Watts})
		}
	}
	return observations
}

func clusterSize(tasks []Task) float64 {
	size := 0.0
	for _, task := range tasks {
		if task.ClassToWatts != nil {
			for _, powerClassWatts := range task.ClassToWatts {
				size += powerClassWatts
			}
		} else {
			size += task.Watts
		}
	}
	return size
}

// information about a cluster of tasks
type TaskCluster struct {
	clusterIndex int
	tasks        []Task
	sizeScore    int // how many other clusters is this one bigger than (in the current workload)
}

// Sorting TaskClusters based on sizeScore
type TaskClusterSorter []TaskCluster

func (slice TaskClusterSorter) Len() int {
	return len(slice)
}

func (slice TaskClusterSorter) Less(i, j int) bool {
	return slice[i].sizeScore <= slice[j].sizeScore
}

func (slice TaskClusterSorter) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

// order clusters in increasing order of task heaviness
// TODO: Make this look into task.ClassToWatts, if present.
func order(clusters map[int][]Task, numberOfClusters int) []TaskCluster {
	// determine the position of the cluster in the ordered list of clusters
	clusterSizeScores := []TaskCluster{}
	for i := 0; i < numberOfClusters; i++ {
		// sizing the current cluster
		sizeI := clusterSize(clusters[i])

		// comparing with the other clusters
		for j := 0; j != i; j++ {
			if sizeI >= clusterSize(clusters[j]) {
				if len(clusterSizeScores) >= i {
					clusterSizeScores[i].sizeScore++
				} else {
					clusterSizeScores[i] = TaskCluster{
						clusterIndex: i,
						tasks:        clusters[i],
						sizeScore:    1,
					}
				}
			}
		}
	}

	// Sorting the clusters based on sizeScore
	sort.Sort(TaskClusterSorter(clusterSizeScores))
	return clusterSizeScores
}

// Classification of Tasks using KMeans clustering using the watts consumption observations
type TasksToClassify []Task

func (tc TasksToClassify) ClassifyTasks(numberOfClusters int) []TaskCluster {
	clusters := make(map[int][]Task)
	observations := getObservations(tc)
	// TODO: Make the number of rounds configurable based on the size of the workload
	if trained, centroids := gokmeans.Train(observations, numberOfClusters, 50); trained {
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

	return order(clusters, numberOfClusters)
}
