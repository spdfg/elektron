#!/usr/bin/env bash
# Copyright (C) 2018 spdfg
#
# This file is part of Elektron.
#
# Elektron is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.
#
# Elektron is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU General Public License for more details.
#
# You should have received a copy of the GNU General Public License
# along with Elektron.  If not, see <http://www.gnu.org/licenses/>.
#

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
