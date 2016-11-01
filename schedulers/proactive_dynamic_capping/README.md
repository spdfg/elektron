##Proactive Dynamic Capping

Perform Cluster wide dynamic capping.

Offer 2 methods:

	1. First Come First Serve -- For each task that needs to be scheduled, in the order in which it arrives, compute the cluster wide cap.
	2. Rank based cluster wide capping -- Sort a given set of tasks to be scheduled, in ascending order of requested watts, and then compute the cluster wide cap for each of the tasks in the ordered set.

main.go contains a set of test functions for the above algorithm.

***.go Files***
*main.go*
```
Contains functions that simulate FCFS and Ranked based scheduling.
```

*task.go*
```
Contains the blue print for a task.
A task contains the following information,
	1. Image -- The image tag of the benchmark.
	2. Name -- The name of the benchmark.
	3. Host -- The host on which the task is to be scheduled.
	4. CMD -- Comamnd to execute the benchmark.
	5. CPU -- CPU shares to be allocated to the task.
	6. RAM -- Amount of RAM to be given to the task.
	7. Watts -- Requested amount of power, in watts.
	8. Inst -- Number of instances.
```

*constants.go*
```
Contains constants that are used by all the subroutines.
Defines the following constants,
	1. Hosts -- The possible hosts on which tasks can be scheduled.
	2. Cap margin -- Margin of the requested power to be given to the task.
	3. Power threshold -- Lower bound of power threshold for a task.
	4. Total power -- Total power (including the static power) per node.
	5. Window size -- size of the window of tasks.
```

*utils.go*
```
Contains functions that are used by all other Go routines.
```

###Please run the following commands to install dependencies and run the test code.
```
	go build
	go run main.go
```

###Note
	The github.com folder contains a library that is required to compute the median of a given set of values.

###Things to do

	1. Need to improve the test cases in main.go.
	2. Need to add more test cases to main.go.
	3. Add better exception handling to capper.go. 
