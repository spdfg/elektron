##Proactive Dynamic Capping

Perform Cluster wide dynamic capping.

Offer 2 methods:

	1. First Come First Serve -- For each task that needs to be scheduled, in the order in which it arrives, compute the cluster wide cap.
	2. Rank based cluster wide capping -- Sort a given set of tasks to be scheduled, in ascending order of requested watts, and then compute the cluster wide cap for each of the tasks in the ordered set.


main.go contains a set of test functions for the above algorithm.

#Please run the following commands to install dependencies and run the test code.
```
	go build
	go run main.go
```

#Note
	The github.com folder contains a library that is required to compute the median of a given set of values.
