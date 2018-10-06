# Scheduling Policy Switch Trace

Every time _Elektron_ switches to a different scheduling policy, the name of the scheduling policy and the time at which the switch took place is logged.

The logs are written to a file named _\<logFilePrefix\>\_\<timestamp\>_schedPolSwitch.log_, where
* _logFilePrefix_ is the prefix provided using the `-logPrefix` option.
* _timestamp_ corresponds to the time when _Elektron_ was run.

The format of the data logged is as shown below.
```
<yyyy/mm/dd> <hh:mm:ss> <scheduling policy name>
```

_Note: The names of the scheduling policies can be found [here](https://gitlab.com/spdf/elektron/blob/master/schedulers/store.go#L14)_

