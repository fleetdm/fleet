FROM golang:1.18.0
ARG TAG
RUN apt update && apt upgrade -y && apt install npm yarnpkg -y && ln -s /usr/bin/yarnpkg /usr/bin/yarn
RUN git clone -b $TAG https://github.com/fleetdm/fleet.git
