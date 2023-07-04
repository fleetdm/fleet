#!/bin/bash
set -e

for ARGUMENT in "$@"
do
   KEY=$(echo $ARGUMENT | cut -f1 -d=)

   KEY_LENGTH=${#KEY}
   VALUE="${ARGUMENT:$KEY_LENGTH+1}"

   export "$KEY"="$VALUE"
done

# Call aws ecs run-task
TASK_ARN="$(aws ecs run-task --region "${REGION}" --cluster "${ECS_CLUSTER}" --task-definition "${TASK_DEFINITION}":"${TASK_DEFINITION_REVISION}" --launch-type FARGATE --network-configuration "awsvpcConfiguration={subnets="${SUBNETS}",securityGroups="${SECURITY_GROUPS}"}" --query 'tasks[].taskArn' --overrides '{"containerOverrides": [{"name": "fleet", "command": ["fleet", "prepare", "db"]}]}' --output text | rev | cut -d'/' -f1 | rev)"

# Wait for completion
aws ecs wait tasks-stopped --cluster="${ECS_CLUSTER}" --tasks="${TASK_ARN}"

# Exit with task's exit code
TASK_EXIT_CODE=$(aws ecs describe-tasks --cluster $ECS_CLUSTER --tasks $TASK_ARN --query "tasks[0].containers[?name=='fleet'].exitCode" --output text)
exit $TASK_EXIT_CODE
