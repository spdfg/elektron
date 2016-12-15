Electron: A power budget manager
======================================

To Do:

 * Create metrics for each task launched [Time to schedule, run time, power used]
 * Have calibration phase?
 * Add ability to use constraints
 * Running average calculations https://en.wikipedia.org/wiki/Moving_average#Exponential_moving_average
 * Make parameters corresponding to each scheduler configurable (possible to have a config template for each scheduler?)
 * Write test code for each scheduler (This should be after the design change)



**Requires Performance-Copilot tool pmdumptext to be installed on the
machine on which electron is launched for logging to work**



How to run (Use the --help option to get information about other command-line options):

`./electron -workload <workload json>`  

*Here, watts would be considered a hard limit when fitting tasks with offers.*

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
      "inst": 9
   },
   {
      "name": "dgemm",
      "cpu": 3.0,
      "ram": 4096,
      "watts": 50,
      "image": "gouravr/dgemm:v2",
      "cmd": "/./mt-dgemm 1024",
      "inst": 9
   }
]
```
