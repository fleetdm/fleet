#!/bin/bash

set -e

function check_for_variable() {
	VARNAME="${1:?}"
	if [ -z "${!VARNAME}" ]; then
		echo -n "Please enter the value for ${VARNAME:?} -=> "
		read ${VARNAME}
		export ${VARNAME}
	fi
}

# Note that this cannot currently run on Darwin ARM64, but maybe
# someday.

case "$(uname)" in
	Darwin)
		SED=gsed
		;;
	Linux)
		SED=sed
		;;
	*)
		echo "Unknown Operating System Unable to Continue"
		exit 1
		;;
esac

# TF_VAR_slack_webhook is redundant, but let's provide a common
# interface.

EXPECTED_VARIABLES=(
	TF_VAR_slack_webhook
	CLOUDFLARE_API_TOKEN
	FLEET_VERSION
)

for VARIABLE in ${EXPECTED_VARIABLES[@]}; do
	check_for_variable "${VARIABLE:?}"
done

FLEET_ECR_REPO="411315989055.dkr.ecr.us-east-2.amazonaws.com"
FLEET_ECR_IMAGE="${FLEET_ECR_REPO:?}/sandbox-prod-eks:${FLEET_VERSION:?}"
FLEET_DOCKERHUB_IMAGE="fleetdm/fleet:${FLEET_VERSION:?}"

pushd "$(dirname ${0})/.."


# Docker Prereqs

aws ecr get-login-password | docker login --username AWS --password-stdin "${FLEET_ECR_REPO:?}"

docker pull "${FLEET_DOCKERHUB_IMAGE:?}"
docker tag "${FLEET_DOCKERHUB_IMAGE:?}" "${FLEET_ECR_IMAGE:?}"
docker push "${FLEET_ECR_IMAGE:?}"

# Update the terraform to deploy FLEET_VERSION.  Requires gsed on Darwin!
${SED:?} -i '/name  = "imageTag"/!b;n;c\    value = "'${FLEET_VERSION:?}'"' PreProvisioner/lambda/deploy_terraform/main.tf

terraform init --backend-config=backend-prod.conf

terraform apply

echo <<-EOTEXT
	Script complete.  Please note this updated PreProvisioner/lambda/deploy_terraform/main.tf
	in order to start using the new version of fleet.

	Please ensure your changes are committed to the repo!
EOTEXT

popd


