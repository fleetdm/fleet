#!/bin/bash

set -x

for (( c=8; c<=240; c+=8 ))
do
        terraform apply -var tag=$BRANCH_NAME -var loadtest_containers=$c -auto-approve
        sleep 400 # let's give some time for hosts to enroll
done
