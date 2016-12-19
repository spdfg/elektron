Electron: Scheduling Algorithms
================================

To Do:

 * Design changes -- Possible to have one scheduler with different scheduling schemes?
 * Use the generic running average calculator in utilities/runAvg in schedulers/proactiveclusterwidecappers.go
 * Add functionality to the runAvg interface to be able to remove element in the window based on a criteria.

Scheduling Algorithms:

 * First Fit
 * First Fit with sorted watts
 * Bin-packing with sorted watts
 * FCFS Proactive Cluster-wide Capping
 * Ranked Proactive Cluster-wide Capping
 * Piston Capping -- Works when scheduler is run with WAR
