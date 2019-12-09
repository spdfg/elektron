# Performance Co-Pilot (PCP) Data

_Elektron_ makes use of PCP to collect performance metrics using the [pcp config](../../config) file.

[pmdumptext](https://pcp.io/man/man1/pmdumptext.1.html) is used to retrieve all the data. The command used to retrieve the performance metrics is shown below (can also be found [here](https://gitlab.com/spdf/elektron/blob/master/pcp/pcp.go#L15)).
```commandline
pmdumptext -m -l -f '' -t 1.0 -d , -c <config file>
```
The logs are written to a file named _\<logFilePrefix\>\_\<timestamp\>.pcplog_, where
* _logFilePrefix_ is the prefix provided using the `-logPrefix` option.
* _timestamp_ corresponds to the time when _Elektron_ was run.

Use `-pminfo` to obtain information about different performance metrics that can be monitored through Performance Co-Pilot. Please see [pminfo doc](https://pcp.io/man/man1/pminfo.1.html) for usage and options.

#### Example PCP log
Assume we want to retrieve the following performance metrics collected from one host, _myhost_.<br>
* System CPU time
* User CPU time

Then the PCP config file would be as shown below.
```
myhost:kernel.all.cpu.user
myhost:kernel.all.cpu.sys
```

When we run the `pmdumptext` command mentioned above for 5 seconds, the PCP log from _Elektron_ would be as shown below.
```
[<loglevel>]: <yyyy-mm-dd> <hh:mm:ss> myhost:kernel.all.cpu.user,myhost:kernel.all.cpu.sys
[<loglevel>]: <yyyy-mm-dd> <hh:mm:ss> <myhost user cpu time>,<myhost system cpu time>
[<loglevel>]: <yyyy-mm-dd> <hh:mm:ss> <myhost user cpu time>,<myhost system cpu time>
[<loglevel>]: <yyyy-mm-dd> <hh:mm:ss> <myhost user cpu time>,<myhost system cpu time>
[<loglevel>]: <yyyy-mm-dd> <hh:mm:ss> <myhost user cpu time>,<myhost system cpu time>
[<loglevel>]: <yyyy-mm-dd> <hh:mm:ss> <myhost user cpu time>,<myhost system cpu time>
```

## Power Measurements
It is also possible to measure the power consumption of CPU, DRAM etc., through the use of RAPL hardware counters.

When running the power capping strategies, [Extrema](../PowerCappingStrategies.md) and [Progressive Extrema](../PowerCappingStrategies.md), the following performance metrics MUST be included in the PCP config file.
```
#RAPL CPU PKG
<hostname1>:perfevent.hwcounters.rapl__RAPL_ENERGY_PKG.value
<hostname2>:perfevent.hwcounters.rapl__RAPL_ENERGY_PKG.value
...
#RAPL DRAM
<hostname1>:perfevent.hwcounters.rapl__RAPL_ENERGY_DRAM.value
<hostname2>:perfevent.hwcounters.rapl__RAPL_ENERGY_DRAM.value
...
```

Note that the power readings are retrieved for each processor on each worker node. For example, if you have two processors on a machine (hostname = _myhost_), then the PCP log for CPU and DRAM power readings would contain the following headers.

`myhost:perfevent.hwcounters.rapl__RAPL_ENERGY_PKG.value["cpux"]`
`myhost:perfevent.hwcounters.rapl__RAPL_ENERGY_PKG.value["cpuy"]`
`myhost:perfevent.hwcounters.rapl__RAPL_ENERGY_DRAM.value["cpux"]`
`myhost:perfevent.hwcounters.rapl__RAPL_ENERGY_DRAM.value["cpuy"]`

Use `-pminfo` and search for RAPL to get the list of RAPL packages from which data can be read from.
