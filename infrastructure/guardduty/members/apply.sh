#!/bin/bash
#last_account=492052055440
#start='false'
for account_id in $(aws organizations list-accounts | jq -r '.Accounts[] | .Id' | grep -v '831217569274' | grep -v '353365949058'); do
    #if [[ ${last_account} == ${account_id} ]]; then
    #    start='true'
    #fi
    #if [[ $start == 'false' ]]; then
    #    continue
    #fi
    for region in $(aws ec2 describe-regions | jq -r '.Regions[] | .RegionName'); do
        terraform workspace new "$account_id:$region"
        terraform workspace select "$account_id:$region" || exit 1
        terraform apply -auto-approve || exit 1
    done
done
