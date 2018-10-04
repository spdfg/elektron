Elektron: A Pluggable Mesos framework with power-aware capabilities
===================================================================

_Elektron_ is a [Mesos](mesos.apache.org) framework that behaves as a playground for developers to experiment with different scheduling policies to schedule ad-hoc jobs in docker containers. It is designed as a lightweight, configurable framework, which can be used in conjunction with built-in power-capping policies to reduce the peak power and/or energy usage of co-scheduled tasks.

However, in addition to being a scheduler, Elektron also takes advantage of tools such as [Performance Co-Pilot](http://pcp.io/) and [RAPL](https://01.org/blogs/2014/running-average-power-limit--rapl) to help contain the power envelope within defined thresholds, reduce peak power consumption, and also reduce total energy consumption. Elektron is able to leverage the Mesos-provided resource abstraction to allow different algorithms to decide how to consume resource offers made by a Mesos Master.

## Architecture
![](docs/ElekArch.png)

_Elektron_ is comprised of three main components: _Task Queue_, _Scheduler_ and _Power Capper_.
* **Task Queue** - Maintains tasks that are yet to be scheduled.
* **Scheduler** - Matches tasks' resource requirements with Mesos resource offers. Tasks that matched offers are then launched on the corresponding nodes.
* **Power Capper** - The Power Capper monitors the power consumption of the nodes in the cluster through the use of [Performance Co-Pilot](http://pcp.io/). A power capping policy uses this information and decides to power cap or power uncap one or more nodes in the cluster using [RAPL](https://01.org/blogs/2014/running-average-power-limit--rapl).

## Usage
* [Pluggable Scheduling Policies](docs/SchedulingPolicies.md)
* [Power-Capping strategies](docs/PowerCappingStrategies.md)
* [Scheduling Policy Switching](docs/SchedulingPolicySwitching.md)

## Logging
Please go through the [logging library doc](docs/Logging.md) to understand how the logging library has been setup. There are also instructions on how one can add additional loggers.

## Data
* [Cluster Resource Consumption](docs/data/ClusterResourceConsumption.md)
* [Schedule Trace](docs/data/ScheduledTrace.md)
* [Degree of Collocation](docs/data/DegreeOfCollocation.md)
* When scheduling policy switching enabled.
    - [Task Classification Overhead](docs/data/withSpsEnabled/TaskClassificationOverhead.md)
    - [Scheduling Policy Switch Trace](docs/data/withSpsEnabled/SchedulingPolicySwitchTrace.md)
    - [Scheduling Window](docs/data/withSpsEnabled/SchedulingWindow.md)

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
How to run (Use the --help option to get information about other command-line options): 

Use the `-logPrefix` option to provide the prefix for the log file names. 

```commandline
./elektron -master <host:port> -workload <workload json>
```

### Power Capping
_Elektron_ is also capable of running power capping policies along with scheduling policies. 

Use the `-powercap` option with the name of the power capping policy to be run.

```commandline
./elektron -master <host:port> -workload <workload json> -powercap <powercap policy name>
```

### Plug-in Scheduling Policy
Use the `-schedPolicy` option with the name of the scheduling policy to be deployed.<br>The default scheduling policy is First Fit.

```commandline
./elektron -master <host:port> -workload <workload json> -schedPolicy <scheduling policy name>
```

_Note_: To obtain the list of possible scheduling policy names, use the `-listSchedPolicies` option.

To run electron with Scheduling Policy Switching Enabled, run the following command,

```commandline
./elektron -master <host:port> -workload <workload json> -ssp -spConfig <schedPolicy config file>
```

Workload schema:

```
[
   {
      "name": "minife",
      "cpu": 3.0,
      "ram": 4096,
      "watts": 63.141,
      "class_to_watts": {
        "A": 93.062,
        "B": 65.552,
        "C": 57.897,
        "D": 60.729
      },
      "image": "rdelvalle/minife:electron1",
      "cmd": "cd src && mpirun -np 3 miniFE.x -nx 100 -ny 100 -nz 100",
      "inst": 10
   },
   {
      "name": "dgemm",
      "cpu": 3.0,
      "ram": 32,
      "watts": 85.903,
      "class_to_watts": {
        "A": 114.789,
        "B": 89.133,
        "C": 82.672,
        "D": 81.944
      },
      "image": "rdelvalle/dgemm:electron1",
      "cmd": "/./mt-dgemm 1024",
      "inst": 10
   }
]
```
