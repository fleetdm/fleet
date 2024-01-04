#!/bin/bash
set -e

function scale_services(){
	UP_DOWN="${1:?}"
	# Set the minimum capacity and desired count in the cluster to 0 to scale down or to the original size to scale back to normal.

	# This is a bit hacky, but the update-service has to happen first when scaling up and second when scaling down.
	# Assume scaling down unless "up".
	CAPACITY=0
	if [ "${UP_DOWN:?}" = "up" ]; then
		aws ecs update-service --region "${REGION:?}" --cluster "${ECS_CLUSTER:?}" --service "${ECS_SERVICE:?}" --desired-count "${DESIRED_COUNT:?}"
		CAPACITY="${MIN_CAPACITY:?}"
	fi
	aws application-autoscaling register-scalable-target --region "${REGION:?}" --service-namespace ecs --resource-id "service/${ECS_CLUSTER:?}/${ECS_SERVICE:?}" --scalable-dimension "ecs:service:DesiredCount" --min-capacity "${CAPACITY:?}"
	# We are scaling down, make it 0
	if [ "${UP_DOWN:?}" != "up" ]; then
		aws ecs update-service --region "${REGION:?}" --cluster "${ECS_CLUSTER:?}" --service "${ECS_SERVICE:?}" --desired-count 0
	fi
	# The first task defintion might never get stable because it never had initial migrations so don't wait before continuing
	if [ "${TASK_DEFINITION_REVISION}" != "1" ]; then
		# Wait for scale-down to succeed
		aws ecs wait services-stable --region "${REGION:?}" --cluster "${ECS_CLUSTER:?}" --service "${ECS_SERVICE:?}"
	fi
}

for ARGUMENT in "$@"
do
   KEY=$(echo $ARGUMENT | cut -f1 -d=)

   KEY_LENGTH=${#KEY}
   VALUE="${ARGUMENT:$KEY_LENGTH+1}"

   export "$KEY"="$VALUE"
done

scale_services down

# Call aws ecs run-task
TASK_ARN="$(aws ecs run-task --region "${REGION:?}" --cluster "${ECS_CLUSTER:?}" --task-definition "${TASK_DEFINITION:?}":"${TASK_DEFINITION_REVISION:?}" --launch-type FARGATE --network-configuration "awsvpcConfiguration={subnets="${SUBNETS:?}",securityGroups="${SECURITY_GROUPS:?}"}" --query 'tasks[].taskArn' --overrides '{"containerOverrides": [{"name": "fleet", "command": ["fleet", "prepare", "db"]}]}' --output text | rev | cut -d'/' -f1 | rev)"

# Wait for completion
aws ecs wait tasks-stopped --region "${REGION:?}" --cluster="${ECS_CLUSTER:?}" --tasks="${TASK_ARN:?}"

scale_services up

# Exit with task's exit code
TASK_EXIT_CODE=$(aws ecs describe-tasks --region "${REGION:?}" --cluster ${ECS_CLUSTER:?} --tasks ${TASK_ARN:?} --query "tasks[0].containers[?name=='fleet'].exitCode" --output text)
exit "${TASK_EXIT_CODE}"
