#!/bin/bash
set -e
# Script for enrolling osquery-perf hosts by `terraform apply`ing in increments of 4 `loadtest` containers.
# NOTE(lucas): This is the currently known configuration that won't tip the loadtest environment,
# but maybe in the future we can be more aggressive (and reduce enroll time).
#
# ./enroll.sh my-branch 8 240

BRANCH_NAME=$1
TASK_SIZE=${2:?}
START_INDEX=$3
END_INDEX=$4
SLEEP_TIME_SECONDS=${5:-60}
INCREMENT=${6:-4}

if ! [[ "$INCREMENT" =~ ^[0-9]+$ ]] || (( INCREMENT <= 0 )); then
	echo "INCREMENT must be a positive integer, got: $INCREMENT"
	exit 1
fi
if [ -z "$BRANCH_NAME" ]; then
	echo "Missing BRANCH_NAME"
fi
if [ -z "$START_INDEX" ]; then
	echo "Missing START_INDEX"
fi
if [ -z "$END_INDEX" ]; then
	echo "Missing END_INDEX"
fi
if [ -z "$TASK_SIZE" ]; then
	echo "Missing TASK_SIZE"
fi

# We add this check to avoid terraform (error-prone) locking in case of typos.
# read -p "You will use BRANCH_NAME=$BRANCH_NAME. Continue? "

set -x

for (( c=$START_INDEX; c<=$END_INDEX; c+=$INCREMENT )); do
    terraform apply -var git_tag_branch=$BRANCH_NAME -var task_size="$TASK_SIZE" -var loadtest_containers=$c -auto-approve
	sleep $SLEEP_TIME_SECONDS
done

# Apply the remainder if the loop didn't land exactly on END_INDEX.
if (( $c - $INCREMENT != $END_INDEX )); then
	terraform apply -var git_tag_branch=$BRANCH_NAME -var task_size="$TASK_SIZE" -var loadtest_containers=$END_INDEX -auto-approve
fi
