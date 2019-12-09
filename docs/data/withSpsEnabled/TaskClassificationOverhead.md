# Task Classification Overhead

To select the next scheduling policy to switch to, the tasks are classified and then the task distribution is determined, as mentioned [here](../../SchedulingPolicySwitching.md). Based on the task distribution, the most appropriate scheduling policy is selected. The time taken to classify the tasks and determine the task distribution is logged as task classification overhead.

The logs are written to a file named _\<logFilePrefix\>\_\<timestamp\>\_classificationOverhead.log_, where
* _logFilePrefix_ is the prefix provided using the `-logPrefix` option.
* _timestamp_ corresponds to the time when _Elektron_ was run.

The format of the data logged is as shown below.
```
[<loglevel>]: <yyyy-mm-dd> <hh:mm:ss> Overhead in microseconds=<overhead>
```

_Note: The classification overhead is logged in **microseconds**._
