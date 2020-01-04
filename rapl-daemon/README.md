# RAPL Daemon

This runs a server that is capable changing the percentage at which
a node is being throttled using RAPL. This daemon should be installed
on all worker nodes.

### Sample payload for testing:
```
 curl --header "Content-Type: application/json" \
  --request POST \
  --data '{"percentage":75}' \
  http://localhost:9090/powercap
 ```
