# RAPL Daemon

This runs a server that is capable of changing the percentage at which
a node is being throttled to using RAPL. This daemon should be installed
on all worker nodes.

### Sample payload for testing:
```
 curl --header "Content-Type: application/json" \
  --request POST \
  --data '{"percentage":75}' \
  http://localhost:9090/powercap
 ```

### Payload

```json
{
    "percentage":75
}
```

### Response

The daemon will respond with a json payload containing zones that were
successfully capped as well as the zones that were not capped.

```json
{
    "cappedZones": null,
    "failedZones": [
        "intel-rapl:0",
        "intel-rapl:1"
    ],
    "error": "some zones were not able to be powercapped"
}
```

Field error will not exist if failed zones is empty.