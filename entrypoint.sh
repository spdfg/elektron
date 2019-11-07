#!/usr/bin/env bash

# Accessing host machine's ip address.
hostip=$HOST_IP

# setting up metrics to be monitored.
# creating PCP config with the cpu and memory usage metrics to be monitored.
cat >config <<EOL
${hostip}:kernel.all.cpu.user
${hostip}:kernel.all.cpu.sys
${hostip}:kernel.all.cpu.idle
${hostip}:mem.util.free
${hostip}:mem.util.used
EOL

./$ELEKTRON_EXECUTABLE_NAME -m $ELEKTRON_MESOS_MASTER_LOCATION -w $ELEKTRON_WORKLOAD -p $ELEKTRON_LOGDIR_PREFIX $@
