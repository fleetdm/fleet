#!/bin/bash

set -x

for (( c=360; c<=408; c+=16 ))
do
        terraform apply -var tag=$BRANCH_NAME -var loadtest_containers=$c -auto-approve
        sleep 300 # let's give some time for hosts to enroll
done
