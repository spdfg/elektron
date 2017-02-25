package def

import (
	"github.com/mdesenfants/gokmeans"
	"sort"
)

// Information about a cluster of tasks
type TaskCluster struct {
	ClusterIndex int
	Tasks        []Task
	SizeScore    int // How many other clusters is this cluster bigger than
}

// Classification of Tasks using KMeans clustering using the watts consumption observations
type TasksToClassify []Task

func (tc TasksToClassify) ClassifyTasks(numberOfClusters int) []TaskCluster {
	clusters := make(map[int][]Task)
	observations := getObservations(tc)
	// TODO: Make the number of rounds configurable based on the size of the workload
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
	return labelAndOrder(clusters, numberOfClusters)
}

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

// Size tasks based on the power consumption
// TODO: Size the cluster in a better way just taking an aggregate of the watts resource requirement.
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

// Order clusters in increasing order of task heaviness
func labelAndOrder(clusters map[int][]Task, numberOfClusters int) []TaskCluster {
	// Determine the position of the cluster in the ordered list of clusters
	sizedClusters := []TaskCluster{}

	// Initializing
	for i := 0; i < numberOfClusters; i++ {
		sizedClusters = append(sizedClusters, TaskCluster{
			ClusterIndex: i,
			Tasks:        clusters[i],
			SizeScore:    0,
		})
	}

	for i := 0; i < numberOfClusters-1; i++ {
		// Sizing the current cluster
		sizeI := clusterSize(clusters[i])

		// Comparing with the other clusters
		for j := i + 1; j < numberOfClusters; j++ {
			sizeJ := clusterSize(clusters[j])
			if sizeI > sizeJ {
				sizedClusters[i].SizeScore++
			} else {
				sizedClusters[j].SizeScore++
			}
		}
	}

	// Sorting the clusters based on sizeScore
	sort.SliceStable(sizedClusters, func(i, j int) bool {
		return sizedClusters[i].SizeScore <= sizedClusters[j].SizeScore
	})
	return sizedClusters
}
