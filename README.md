Elektron: A Pluggable Mesos framework with power-aware capabilities
===================================================================

_Elektron_ is a [Mesos](mesos.apache.org) framework that behaves as a playground for developers to experiment with different scheduling policies to schedule ad-hoc jobs in docker containers. It is designed as a lightweight, configurable framework, which can be used in conjunction with built-in power-capping policies to reduce the peak power and/or energy usage of co-scheduled tasks.

However, in addition to being a scheduler, Elektron also takes advantage of tools such as [Performance Co-Pilot](http://pcp.io/) and [RAPL](https://01.org/blogs/2014/running-average-power-limit--rapl) to help contain the power envelope within defined thresholds, reduce peak power consumption, and also reduce total energy consumption. Elektron is able to leverage the Mesos-provided resource abstraction to allow different algorithms to decide how to consume resource offers made by a Mesos Master.

## Architecture
![](docs/ElekArch.png)

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

`./elektron -workload <workload json>`

To run electron with Scheduling Policy Switching Enabled, run the following command,

`./electron -workload <workload json> -ssp -spConfig <schedPolicy config file>`

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
