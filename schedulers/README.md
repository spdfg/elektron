Electron: Scheduling Algorithms
================================

To Do:

 * Design changes -- Possible to have one scheduler with different scheduling schemes?
 * Fix the race condition on 'tasksRunning' in proactiveclusterwidecappingfcfs.go and proactiveclusterwidecappingranked.go
 * Separate the capping strategies from the scheduling algorithms and make it possible to use any capping strategy with any scheduler.

Scheduling Algorithms:

 * First Fit
 * First Fit with sorted watts
 * Bin-packing with sorted watts
 * FCFS Proactive Cluster-wide Capping
 * Ranked Proactive Cluster-wide Capping
 * Piston Capping -- Works when scheduler is run with WAR
