#!/bin/bash
set -e

function scale_services(){
	UP_DOWN="${1:?}"
	SERVICE_NAME="${2:?}" # Take service name as an argument
	ADJUST_AUTOSCALING="${3:-}"
	COUNT="${4:-1}"

	# Set the minimum capacity and desired count in the cluster to 0 to scale down or to the original size to scale back to normal.

	# This is a bit hacky, but the update-service has to happen first when scaling up and second when scaling down.
	# Assume scaling down unless "up".
	CAPACITY=0
	if [ "${UP_DOWN:?}" = "up" ]; then
		aws ecs update-service --region "${REGION:?}" --cluster "${ECS_CLUSTER:?}" --service "${SERVICE_NAME:?}" --desired-count "${COUNT:?}"
		CAPACITY="${MIN_CAPACITY:?}"
	fi

  if [ -n "${ADJUST_AUTOSCALING}" ]; then
    aws application-autoscaling register-scalable-target --region "${REGION:?}" --service-namespace ecs --resource-id "service/${ECS_CLUSTER:?}/${SERVICE_NAME:?}" --scalable-dimension "ecs:service:DesiredCount" --min-capacity "${CAPACITY:?}"
  fi
	# We are scaling down, make it 0
	if [ "${UP_DOWN:?}" != "up" ]; then
		aws ecs update-service --region "${REGION:?}" --cluster "${ECS_CLUSTER:?}" --service "${SERVICE_NAME:?}" --desired-count 0
	fi
	# The first task definition might never get stable because it never had initial migrations so don't wait before continuing
	if [ "${TASK_DEFINITION_REVISION}" != "1" ]; then
		# Wait for scale-down to succeed
		aws ecs wait services-stable --region "${REGION:?}" --cluster "${ECS_CLUSTER:?}" --services "${SERVICE_NAME:?}"
	fi
}

for ARGUMENT in "$@"
do
   KEY=$(echo $ARGUMENT | cut -f1 -d=)
   KEY_LENGTH=${#KEY}
   VALUE="${ARGUMENT:$KEY_LENGTH+1}"
   export "$KEY"="$VALUE"
done

scale_services down "${ECS_SERVICE:?}" true "${DESIRED_COUNT}"

if [ -n "${VULN_SERVICE}" ]; then
  scale_services down "${VULN_SERVICE:?}"
fi

# Call aws ecs run-task
TASK_ARN="$(aws ecs run-task --region "${REGION:?}" --cluster "${ECS_CLUSTER:?}" --task-definition "${TASK_DEFINITION:?}":"${TASK_DEFINITION_REVISION:?}" --launch-type FARGATE --network-configuration "awsvpcConfiguration={subnets="${SUBNETS:?}",securityGroups="${SECURITY_GROUPS:?}"}" --query 'tasks[].taskArn' --overrides '{"containerOverrides": [{"name": "fleet", "command": ["fleet", "prepare", "db"]}]}' --output text | rev | cut -d'/' -f1 | rev)"

# Wait for completion
aws ecs wait tasks-stopped --region "${REGION:?}" --cluster="${ECS_CLUSTER:?}" --tasks="${TASK_ARN:?}"

scale_services up "${ECS_SERVICE:?}" true "${DESIRED_COUNT}"

if [ -n "${VULN_SERVICE}" ]; then
  scale_services up "${VULN_SERVICE:?}"
fi

# Exit with task's exit code
TASK_EXIT_CODE=$(aws ecs describe-tasks --region "${REGION:?}" --cluster "${ECS_CLUSTER:?}" --tasks "${TASK_ARN:?}" --query "tasks[0].containers[?name=='fleet'].exitCode" --output text)
exit "${TASK_EXIT_CODE}"
