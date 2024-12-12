#!/bin/bash

# Script for enrolling osquery-perf hosts by `terraform apply`ing in increments of 8 `loadtest` containers.
# NOTE(lucas): This is the currently known configuration that won't tip the loadtest environment,
# but maybe in the future we can be more aggressive (and reduce enroll time).
#
# ./enroll.sh my-branch 8 240

BRANCH_NAME=$1
START_INDEX=$2
END_INDEX=$3
INCREMENT=8
SLEEP_TIME_SECONDS=60

if [ -z "$BRANCH_NAME" ]; then
	echo "Missing BRANCH_NAME"
	exit 1
fi
if [ -z "$START_INDEX" ]; then
	echo "Missing START_INDEX"
	exit 1
fi
if [ -z "$END_INDEX" ]; then
	echo "Missing END_INDEX"
	exit 1
fi

# We add this check to avoid terraform (error-prone) locking in case of typos.
read -p "You will use BRANCH_NAME=$BRANCH_NAME. Continue? (y/N) "
if [ "$REPLY" != "y" ]; then echo ok, bailing out for safety...; exit 1; fi

set -x

for (( c=$START_INDEX; c<=$END_INDEX; c+=$INCREMENT )); do
	terraform apply -var tag=$BRANCH_NAME -var loadtest_containers=$c -auto-approve
	sleep $SLEEP_TIME_SECONDS
done
