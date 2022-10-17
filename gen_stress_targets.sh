#! /bin/bash

for i in {1..10}
do
    `curl -X GET 'https://localhost:8080/api/fleet/stress/scenario' --insecure | jq '.' > stress_targets/${i}.json`
    `echo "POST https://localhost:8080/api/fleet/stress/trial" >> 'stress_targets/target.list'`
    `echo "Content-Type: application/json" >> 'stress_targets/target.list'`
    `echo "@${i}.json" >> 'stress_targets/target.list'`
    `echo "" >> 'stress_targets/target.list'`
done