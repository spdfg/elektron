## Scheduling Policy Switching

When scheduling the workload, _Elektron_ can be made to switch between different scheduling policies in order to adapt to the variation in the workload and the changes in the state of the cluster.

### Scheduling Policy Configuration File
A scheduling policy configuration (SPConfig) file needs to be provided that contains information regarding the scheduling policies that can be selected while switching and the distribution of tasks that they are appropriate to schedule (_See [schedPolConfig.json](../schedPolConfig.json) for reference_).

```json
{
    "bin-packing": {
        "taskDist": 10.0
    },
    "max-greedymins": {
        "taskDist": 6.667
    },
    "max-min": {
        "taskDist": 0.416
    },
    "first-fit": {
        "taskDist": 0.05
    }
}
```

### Scheduling Policy Selector
![](docs/SchedPolSelector.png)

The **Scheduling Policy Selector** is responsible for selecting the appropriate scheduling policy to schedule the next set of pending tasks.

It takes as input a _Switching Criteria_, the pending task queue and Mesos resource offers. The Scheduling Policy Selector is made up of the following components.
* **_Resource Availability Tracker_** - Tracks the clusterwide availability of compute resources such as CPU and memory using Mesos resource offers.
* **_Window Calculator_** - Determines the scheduling window based on the cluster resource availability and the set of pending tasks. This window constitutes the set of pending tasks that the next policy schedules.
* **_Policy Selector_** - Based on the _Switching Criteria_, the policy selector selects the next appropriate scheduling policy. The default _Switching Criteria_ is task distribution based. However, _Elektron_ also supports round-robin based switching.
    * _Task Distribution based switching_ - The tasks in the scheduling window are first classified (based on their estimated power consumption) into Low Power Consuming (L<sub>pc</sub>) and High Power Consuming (H<sub>pc</sub>). The distribution of tasks is then determined to be the ratio of the number of L<sub>pc</sub> tasks and H<sub>pc</sub> tasks. The _Policy Selector_ then selects the scheduling policy that is most appropriate to schedule the determined distribution of tasks (using information in SPConfig file).
    * _Round-Robin based switching_ - The _Policy Selector_ selects the next scheduling policy based on a round-robin ordering. For this, the scheduling policies mentioned in SPConfig are stored in a non-increasing order of their corresponding task distribution.
