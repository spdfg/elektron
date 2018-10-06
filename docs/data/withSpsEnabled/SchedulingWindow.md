# Scheduling Window

Every time _Elektron_ switches to a different scheduling policy, the size of the scheduling window and the name of the scheduling policy is logged.

The logs are written to a file named _\<logFilePrefix\>\_\<timestamp\>\_schedWindow.log_, where
* _logFilePrefix_ is the prefix provided using the `-logPrefix` option.
* _timestamp_ corresponds to the time when _Elektron_ was run.

The format of the data logged is as shown below.
```
<yyyy/mm/dd> <hh:mm:ss> <sched window size> <sched policy name>
```