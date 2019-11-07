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

FROM ubuntu:xenial

RUN apt-get update && apt-get -y install \
    apt-transport-https \
    ca-certificates \
    curl \
    software-properties-common && \
    apt-get install -yq ssh git

RUN add-apt-repository ppa:deadsnakes/ppa && \
	apt-get update && \
	apt-get install -y python3.7

# Installing performance co-pilot.
RUN apt-get install -y pcp pcp-gui

# The name of the elektron executable.
ARG elektronexecutablename=elektron
ENV ELEKTRON_EXECUTABLE_NAME=$elektronexecutablename

# HOST:PORT of the mesos master.
ARG elektronmesosmasterlocation=localhost:5050
ENV ELEKTRON_MESOS_MASTER_LOCATION=$elektronmesosmasterlocation

# Workload to be scheduled.
ARG elektronworkload=workload_sample.json
ENV ELEKTRON_WORKLOAD=$elektronworkload

# Prefix of the log directory.
ARG elektronlogdirprefix=Elektron-Test-Run
ENV ELEKTRON_LOGDIR_PREFIX=$elektronlogdirprefix

# Hostname/IP Address of the physical machine on which the docker containers are being run.
# This is used to create the pcp config file.
ARG hostip=""
ENV HOST_IP=$hostip

# Creating directory into which the current directory is going to be mounted.
RUN mkdir /elektron
ADD ./ /elektron
WORKDIR /elektron
RUN chmod 777 /elektron/entrypoint.sh

ENTRYPOINT ["/elektron/entrypoint.sh"]
