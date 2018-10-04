# Pluggable Scheduling Policies

_Elektron_ is capable of deploying different scheduling policies. The scheduling policy to be plugged in can be specified using the `-schedPolicy` command-line option.

## Scheduling Policies

 * **First Fit** - *Find the first task in the job queue whose 
 resource constraints are satisfied by the resources available in
 a resource offer. If a match is made between an offer and a task, 
 the offer is consumed in order to schedule the matching task. Else,
 move onto a new mesos resource offer and repeat the process.*
 * **Bin-packing** - *For each resource offer received, tasks are matched
 from the priority queue keyed by the Watts resource requirement. If a task's
 resource requirements are not satisfied by the remaining resources in the 
 offer, the next task in the queue is evaluated for fitness. This way, tasks
 are packed into a resource offer.*
 * **Max-Min** - *Pack a resource offer by picking a mixed set of tasks 
 from the job queue (used as a double-ended queue) that is sorted in 
 non-decreasing order of tasks' Watts requirement. Max-Min alternates
 between picking tasks from the front and back of the queue.*
 * **Max-GreedyMins** - *Similar to Max-Min, a double-ended queue,
 sorted in non-decreasing order of tasks' Watts requirement, to store
 the tasks. Max-GreedyMins aims to pack tasks into an offer by picking
 one task from the end of the queue, and as many from the beginning, 
 until no more task can fit the offer.*
