Electron: A power budget manager
======================================

To Do:

 * Create metrics for each task launched [Time to schedule, run time, power used]
 * Have calibration phase?
 * Add ability to use constraints
 * Running average calculations https://en.wikipedia.org/wiki/Moving_average#Exponential_moving_average
 * Make parameters corresponding to each scheduler configurable (possible to have a config template for each scheduler?)
 * Adding type of scheduler to be used, to be picked from a config file, along with it's configurable parameters.
 * Write test code for each scheduler (This should be after the design change)
   Possible to setup the constants at runtime based on the environment?
 * Log fix for declining offer -- different reason when insufficient resources as compared to when there are no
    longer any tasks to schedule.
 * Have a centralised logFile that can be filtered by identifier. All electron logs should go into this file.
 * Make def.Task an interface for further modularization and flexibility.
 * Convert def#WattsToConsider(...) to be a receiver of def.Task and change the name of it to Watts(...).
 * Have a generic sorter for task resources instead of having one for each kind of resource.

**Requires [Performance Co-Pilot](http://pcp.io/) tool pmdumptext to be installed on the
machine on which electron is launched for logging to work and PCP collector agents installed
on the Mesos Agents**


How to run (Use the --help option to get information about other command-line options):

`./electron -workload <workload json>`

To run electron with Watts as Resource, run the following command,

`./electron -workload <workload json> -wattsAsAResource`


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
