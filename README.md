Electron: A power budget manager
======================================

To Do:

 * Create metrics for each task launched [Time to schedule, run time, power used]
 * Have calibration phase?
 * Add ability to use constraints
 * Running average calculations https://en.wikipedia.org/wiki/Moving_average#Exponential_moving_average
 * Make parameters corresponding to each scheduler configurable (possible to have a config template for each scheduler?)
 * Write test code for each scheduler (This should be after the design change)
 * Some of the constants in constants/constants.go can vary based on the environment.  
   Possible to setup the constants at runtime based on the environment?
 * Retrofit schedulers for scheduling tracing


**Requires [Performance Co-Pilot](http://pcp.io/) tool pmdumptext to be installed on the
machine on which electron is launched for logging to work and PCP collector agents installed
on the Mesos Agents**


How to run (Use the --help option to get information about other command-line options):

`./electron -workload <workload json>`

To run electron with ignoreWatts, run the following command,

`./electron -workload <workload json> -ignoreWatts`


Workload schema:

```
[
   {
      "name": "minife",
      "cpu": 3.0,
      "ram": 4096,
      "watts": 50,
      "image": "gouravr/minife:v5",
      "cmd": "cd src && mpirun -np 1 miniFE.x -nx 100 -ny 100 -nz 100",
      "inst": 9,
      "class_to_watts" : {
          "A": 30.2475289996,
          "B": 35.6491229228,
          "C": 24.0476734352
       }

   },
   {
      "name": "dgemm",
      "cpu": 3.0,
      "ram": 4096,
      "watts": 50,
      "image": "gouravr/dgemm:v2",
      "cmd": "/./mt-dgemm 1024",
      "inst": 9,
      "class_to_watts" : {
          "A": 35.2475289996,
          "B": 25.6491229228,
          "C": 29.0476734352
       }
   }
]
```
