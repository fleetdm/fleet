#!/bin/bash
for region in $(aws ec2 describe-regions | jq -r '.Regions[] | .RegionName'); do
    terraform workspace select $region || break
    terraform apply -auto-approve || break
done
