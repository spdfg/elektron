Elektron: A Pluggable Mesos framework with power-aware capabilities
===================================================================

[![Build Status](https://gitlab.com/spdf/elektron/badges/master/build.svg)](https://gitlab.com/spdf/elektron/commits/master)

_Elektron_ is a [Mesos](mesos.apache.org) framework that behaves as a playground for developers to experiment with different scheduling policies to schedule ad-hoc jobs in docker containers. It is designed as a lightweight, configurable framework, which can be used in conjunction with built-in power-capping policies to reduce the peak power and/or energy usage of co-scheduled tasks.

However, in addition to being a scheduler, Elektron also takes advantage of tools such as [Performance Co-Pilot](http://pcp.io/) and [RAPL](https://01.org/blogs/2014/running-average-power-limit--rapl) to help contain the power envelope within defined thresholds, reduce peak power consumption, and also reduce total energy consumption. Elektron is able to leverage the Mesos-provided resource abstraction to allow different algorithms to decide how to consume resource offers made by a Mesos Master.

## Architecture
![](docs/ElekArch.png)

_Elektron_ is comprised of three main components: _Task Queue_, _Scheduler_ and _Power Capper_.
* **Task Queue** - Maintains tasks that are yet to be scheduled.
* **Scheduler** - Matches tasks' resource requirements with Mesos resource offers. Tasks that matched offers are then launched on the corresponding nodes.
* **Power Capper** - The Power Capper monitors the power consumption of the nodes in the cluster through the use of [Performance Co-Pilot](http://pcp.io/). A power capping policy uses this information and decides to power cap or power uncap one or more nodes in the cluster using [RAPL](https://01.org/blogs/2014/running-average-power-limit--rapl).

## Published Research using Elektron
* Pradyumna Kaushik, Akash Kothawale, Renan DelValle, Abhishek Jain, Madhusudhan Govindaraju, “Analysis of Dynamically Switching Energy-Aware Scheduling Policies for Varying Workloads”, in the 11th IEEE International Conference on Cloud Computing (IEEE Cloud), 2018. [[pdf](http://cloud.cs.binghamton.edu/wordpress/wp-content/uploads/2018/05/analysis-of-energy-aware-scheduling-policy-switching-for-varying-workloads.pdf)]
* Renan Delvalle, Pradyumna Kaushik, Abhishek Jain, Jessica Hartog, Madhusudhan Govindaraju, “Exploiting Efficiency Opportunities Based on Workloads with Electron on Heterogeneous Clusters”, in the The 10th IEEE/ACM International Conference on Utility and Cloud Computing (UCC 2017), 2017. [[pdf](http://cloud.cs.binghamton.edu/wordpress/wp-content/uploads/2017/10/ExploitingEfficiencyOpportunitiesBasedonWorkloadswithElectrononHeterogeneousClusters.pdf)]
* Renan DelValle, Abhishek Jain, Pradyumna Kaushik, Jessica Hartog, Madhusudhan Govindaraju, “Electron: Towards Efficient Resource Management on Heterogeneous Clusters with Apache Mesos”, in the IEEE International Conference on Cloud Computing (CLOUD), Applications Track, 2017. [[pdf](http://cloud.cs.binghamton.edu/wordpress/wp-content/uploads/2017/07/electron-1.pdf)]

Note: Elektron was previously known as Electron. We decided to change the name of the framework to avoid confusion with other projects named Electron.

## Features
* [Pluggable Scheduling Policies](docs/SchedulingPolicies.md)
* [Pluggable Power-Capping strategies](docs/PowerCappingStrategies.md)
* [Scheduling Policy Switching](docs/SchedulingPolicySwitching.md)

## Logs
Please go through the [log info](docs/Logs.md) to get information on different data that are logged.

## Software Requirements
**Requires [Performance Co-Pilot](http://pcp.io/) tool pmdumptext to be installed on the
machine on which electron is launched for logging to work and PCP collector agents installed
on the Mesos Agents**

Compatible with the following versions:

* Mesos 1.5.0
* Go 1.9.7

## Build and Run
Compile the source code using the `go build` tool as shown below.
```commandline
go build -o elektron
```
Use the `-h` option to get information about other command-line options.

### Workload
Use the `-workload` option to specify the location of the workload json file. Below is an example workload.
```json
[
   {
      "name": "minife",
      "cpu": 3.0,
      "ram": 4096,
      "watts": 63.141,
      "image": "rdelvalle/minife:electron1",
      "cmd": "cd src && mpirun -np 3 miniFE.x -nx 100 -ny 100 -nz 100",
      "inst": 10
   },
   {
      "name": "dgemm",
      "cpu": 3.0,
      "ram": 32,
      "watts": 85.903,
      "image": "rdelvalle/dgemm:electron1",
      "cmd": "/./mt-dgemm 1024",
      "inst": 10
   }
]
```

```commandline
./elektron -master <host:port> -workload <workload json>
```

Use the `-logPrefix` option to provide the prefix for the log file names.

### Plug-in Power Capping
_Elektron_ is also capable of running power capping policies along with scheduling policies. 

Use the `-powercap` option with the name of the power capping policy to be run.

```commandline
./elektron -master <host:port> -workload <workload json> -powercap <powercap policy name>
```

If the power capping policy is _Extrema_ or _Progressive Extrema_, then the following options must also be specified.
* `-hiThreshold` - If the average historical power consumption of the cluster exceeds this value, then one or more nodes would be power capped.
* `-loThreshold` - If the average historical power consumption of the cluster is lesser than this value, then one or more nodes would be uncapped.

### Plug-in Scheduling Policy
Use the `-schedPolicy` option with the name of the scheduling policy to be deployed.<br>The default scheduling policy is First Fit.

```commandline
./elektron -master <host:port> -workload <workload json> -schedPolicy <sched policy name>
```

_Note_: To obtain the list of possible scheduling policy names, use the `-listSchedPolicies` option.

### Enable Scheduling Policy Switching
Use the `-switchSchedPolicy` option to enable scheduling policy switching.<br>

One needs to also provide a scheduling policy configuration file (see [schedPolConfig](./schedPolConfig.json) for reference).<br>
Use the `-schedPolConfig` option to specify the path of the scheduling policy configuration file.

```commandline
./elektron -master <host:port> -workload <workload json> -switchSchedPolicy -schedPolConfig <config file>
```

The following options can be used when scheduling policy switching is enabled.
* `-fixFirstSchedPol` - Fix the first scheduling policy that is deployed. 
* `-fixSchedWindow` - Allow the size of the scheduling window to be fixed.
* `-schedWindowSize` - Specify the size of the scheduling window. If no scheduling window size specified and `fixSchedWindow` option is enabled, the default size of 200 is used.
* `-schedPolSwitchCriteria` - Criteria to be used when deciding the next scheduling policy to switch to. Default criteria is task distribution (_taskDist_) based. However, one can either switch based on a Round Robin (_round-robin_) or Reverse Round Robin (_rev-round-robin_) order.
