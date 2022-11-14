#!/bin/bash

set -e

function get_unclaimed_instances() {
	aws dynamodb scan --table-name sandbox-prod-lifecycle | jq -r '.Items[] | select(.State.S == "unclaimed") | .ID.S'
}

function purge_instances() {
	INSTANCES="${1}"
	for INSTANCE in ${INSTANCES}; do
		# set -e should force this to abort on any error
		terraform workspace select "${INSTANCE:?}"
		terraform apply -destroy -auto-approve
		terraform workspace select default
		terraform workspace delete "${INSTANCE:?}"
	done	
}

function provision_new_instances() {
	echo "Running ${PREPROVISIONER_TASK_DEFINITION_ARN:?}"
	TASK_ARN="$(aws ecs run-task --region us-east-2 --cluster sandbox-prod --task-definition "${PREPROVISIONER_TASK_DEFINITION_ARN:?}" --launch-type FARGATE --network-configuration 'awsvpcConfiguration={subnets="subnet-055269a06c5204d20",securityGroups="sg-0f7fb24be3617d79c"}' | jq -r '.tasks[0].taskArn')"
	while : ; do
		# Wait at least 60 seconds before checking on status to allow
		# time for it to spin up in FARGATE.
		sleep 60
		TASK_STATUS="$(aws ecs describe-tasks --tasks "${TASK_ARN:?}" --cluster sandbox-prod | jq -r '.tasks[0].desiredStatus')"
		echo "${TASK_ARN:?} status is currently ${TASK_STATUS:?}"
		if [ "${TASK_STATUS:?}" = "STOPPED" ]; then
			break
		fi
	done
}

cat <<-EOWARN
	WARNING:

	You must be logged into the AWS CLI _and_ the VPN for this to work!

	Please note that in order to upgrade the running image or the included standard
	query library, the terraform updating the task definition should be run prior
	to running this script!  You will also need to push the appropriate fleetdm/fleet
	image to ECR.

	Press ENTER to continue or CTRL+C to abort.
EOWARN
read 

pushd "$(dirname "${0}")/../JITProvisioner/deprovisioner/deploy_terraform"

export TF_VAR_eks_cluster="sandbox-prod"
export TF_VAR_mysql_secret="arn:aws:secretsmanager:us-east-2:411315989055:secret:/fleet/database/password/mysql-boxer-QGmEeA"

terraform init -backend-config=backend.conf

PREPROVISIONER_TASK_DEFINITION_ARN="$(aws ecs list-task-definitions | jq -r '.taskDefinitionArns[] | select(contains("sandbox-prod-preprovisioner"))')"
UNCLAIMED_INSTANCES="$(get_unclaimed_instances)"

# From the 11th and on.
ALL_BUT_TEN="$(tail -n +11 <<<"${UNCLAIMED_INSTANCES}")"

purge_instances "${ALL_BUT_TEN:?}"

provision_new_instances

REMAINING_UNCLAIMED="$(comm -12 <(get_unclaimed_instances) <(echo "${UNCLAIMED_INSTANCES:?}"))"

purge_instances "${REMAINING_UNCLAIMED:?}"

provision_new_instances

popd
