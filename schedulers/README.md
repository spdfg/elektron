Electron: Scheduling Algorithms
================================

To Do:

 * Design changes -- Possible to have one scheduler with different scheduling schemes?
 * Make the running average calculation generic, so that schedulers in the future can use it and not implement their own.

Scheduling Algorithms:

 * First Fit
 * First Fit with sorted watts
 * Bin-packing with sorted watts
 * FCFS Proactive Cluster-wide Capping
 * Ranked Proactive Cluster-wide Capping
 * Piston Capping -- Works when scheduler is run with WAR
