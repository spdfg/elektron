# Scheduled Trace

For every task that is scheduled, the task ID and the hostname of the node on which it was 
launched is logged.

The scheduled trace logs are written to a file named _\<logFilePrefix\>\_\<timestamp\>_schedTrace.log_, where
* _logFilePrefix_ is the prefix provided using the `-logPrefix` option.
* _timestamp_ corresponds to the time when _Elektron_ was run.

The format of the data logged is as shown below.
```
<yyyy/mm/dd> <hh:mm:ss> <hostname>:<task ID>
```

