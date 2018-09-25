Elektron: A Pluggable Mesos framework with power-aware capabilities
===================================================================

_Elektron_ is a Mesos framework that behaves as a playground for developers to experiment with different scheduling policies to launch ad-hoc jobs. It is designed as a lightweight, configurable framework, which can be used in conjunction with built-in power-capping policies to reduce the peak power and/or energy usage of co-scheduled tasks.

_Elektron_ meets the need for a more fine-grained control in
order to make informed choices regarding resources offered
by Mesos. It acts as a light-weight framework capable of
scheduling tasks in Docker containers on Mesos. However, in
addition to being a scheduler, Elektron also takes advantage
of tools such as [Performance Co-Pilot](http://pcp.io/) and [RAPL](https://01.org/blogs/2014/running-average-power-limit--rapl) to help contain the power envelope within defined thresholds, reduce peak power consumption, and also reduce total energy consumption. Electron is able to leverage the Mesos-provided resource abstraction to allow different algorithms to decide how to consume resource offers made by a Mesos Master.

#Features
* [Pluggable Scheduling Policies](docs/SchedulingPolicies.md)
* [Pluggable Power-Capping strategies](docs/PowerCappingStrategies.md)
* Cluster resource monitoring

#Software Requirements
**Requires [Performance Co-Pilot](http://pcp.io/) tool pmdumptext to be installed on the
machine on which electron is launched for logging to work and PCP collector agents installed
on the Mesos Agents**

Compatible with the following versions,

* Mesos 1.5.0
* Go 1.9.7

#Build and Run
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
