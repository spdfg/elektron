Electron: Scheduling Algorithms
================================

To Do:

 * Design changes -- Possible to have one scheduler with different scheduling schemes?
 * Fix the race condition on 'tasksRunning' in proactiveclusterwidecappingfcfs.go and proactiveclusterwidecappingranked.go
 * **Critical**: Separate the capping strategies from the scheduling algorithms and make it possible to use any capping strategy with any scheduler.
 * Create a package that would contain routines to perform various logging and move helpers.coLocated(...) into that.
 * Refine the sorting algorithm that sorts the clusters of tasks retrieved using the kmeans algorithm. This also involves the reduction in time complexity of the same.

Scheduling Algorithms:

 * First Fit
 * First Fit with sorted watts
 * Bin-packing with sorted watts
 * BinPacked-MaxMin -- Packing one large task with a bunch of small tasks.
 * Top Heavy -- Hybrid scheduler that packs small tasks (less power intensive) using Bin-packing and spreads large tasks (power intensive) using First Fit.
 * Bottom Heavy -- Hybrid scheduler that packs large tasks (power intensive) using Bin-packing and spreads small tasks (less power intensive) using First Fit. 
 
 Capping Strategies
 
 * Extrema Dynamic Capping
 * Proactive Cluster-wide Capping
 * Piston Capping
