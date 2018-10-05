# Logging

_Elektron_ logs can be categorized into the following.
* _**Console Logs**_ - These are logs written to the console. These include logs related to the Mesos resource offers, offer matching, declining offers and task status updates. They are of the following types.
    * **ERROR**
    * **WARNING**
    * **SUCCESS**
    * **GENERAL**
* [**_Schedule Trace Logs_ (SCHED\_TRACE)**](data/ScheduledTrace.md) - Once each task has fit into a resource offer, the taskID and the corresponding hostname are logged.
* [**PCP**](data/PCP.md) - For every second, data related to load, resource utilization, power consumption etc., is logged. The metrics to be logged need to be specified in the [PCP config file](../config).
* _**Scheduling Policy Switching Logs**_ - When scheduling policy switching is enabled (`-switchSchedPol` is used when launching _Elektron_), the following information is logged.
    * [**Scheduling Policy Switch trace (SPS)**](data/withSpsEnabled/SchedulingPolicySwitchTrace.md) - Every time _Elektron_ switches to a different scheduling policy, the _name_ of the scheduling policy and the corresponding _time stamp_ is logged.<br>
    * [**SCHED_WINDOW**](data/withSpsEnabled/SchedulingWindow.md) - For every switch, the size of the scheduling window and the name of the scheduling policy is logged.
    * [**CLSFN_TASKDIST_OVERHEAD**](data/withSpsEnabled/TaskClassificationOverhead.md) - If the switching criteria is task distribution based, then the time taken to classify the tasks into low power consuming and high power consuming, and then to determine the task distribution is logged.
