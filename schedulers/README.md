Electron: Scheduling Algorithms
================================

To Do:

 * Design changes -- Possible to have one scheduler with different scheduling schemes?
 * Fix the race condition on 'tasksRunning' in proactiveclusterwidecappingfcfs.go and proactiveclusterwidecappingranked.go
 * Separate the capping strategies from the scheduling algorithms and make it possible to use any capping strategy with any scheduler.
 * Make newTask(...) variadic where the newTaskClass argument can either be given or not. If not give, then pick task.Watts as the watts attribute, else pick task.ClassToWatts[newTaskClass].
 * Retrofit pcp/proactiveclusterwidecappers.go to include the power capping go routines and to cap only when necessary.
 * Create a package that would contain routines to perform various logging and move helpers.coLocated(...) into that.

Scheduling Algorithms:

 * First Fit
 * First Fit with sorted watts
 * Bin-packing with sorted watts
 * FCFS Proactive Cluster-wide Capping
 * Ranked Proactive Cluster-wide Capping
 * Piston Capping -- Works when scheduler is run with WAR
 * ClassMapWatts -- Bin-packing and First Fit that now use Watts per power class.
 * Top Heavy -- Hybrid scheduler that packs small tasks (less power intensive) using Bin-packing and spreads large tasks (power intensive) using First Fit.
 * Bottom Heavy -- Hybrid scheduler that packs large tasks (power intensive) using Bin-packing and spreads small tasks (less power intensive) using First Fit. 
