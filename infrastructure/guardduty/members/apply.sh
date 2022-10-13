#!/bin/bash
for account_id in 
    160035666661
do
    for region in $(aws ec2 describe-regions | jq -r '.Regions[] | .RegionName'); do
        terraform workspace new $region
        terraform workspace select $region || break
        terraform apply -auto-approve -var="account_id=${account_id}" || break
    done
done
