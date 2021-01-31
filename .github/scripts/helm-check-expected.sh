#!/usr/bin/env bash

set -eou pipefail

usage() {
    echo "${0} <templated-deployment.yaml> 'LIST OF EXPECTED ENVS' 'LIST OF EXPECTED VOLUMES'"
}

TEMPLATED_DEPLOYMENT=${1}
if [ -z "${TEMPLATED_DEPLOYMENT}" ]; then
    echo "Error: Missing path to templated deployment"
    usage
    exit 1
fi

EXPECTED_ENVS=${2}
if [ -z "${EXPECTED_ENVS}" ]; then
    echo "Error: Missing space-separated list of expected environment variables"
    usage
    exit 1
fi

EXPECTED_VOLUMES=${3}
if [ -z "${EXPECTED_VOLUMES}" ]; then
    echo "Error: Missing space-separated list of expected volumes"
    usage
    exit 1
fi

ALL_ENVS=$(yq eval 'select(.kind == "Deployment") | .spec.template.spec.containers[].env[].name' ${TEMPLATED_DEPLOYMENT})
ALL_VOLUME_MOUNTS=$(yq eval 'select(.kind == "Deployment") | .spec.template.spec.containers[].volumeMounts[].name' ${TEMPLATED_DEPLOYMENT})
ALL_VOLUMES=$(yq eval 'select(.kind == "Deployment") | .spec.template.spec.volumes[].name' ${TEMPLATED_DEPLOYMENT})

seen=0
for EE in ${EXPECTED_ENVS}; do
    seen=0
    for AE in ${ALL_ENVS}; do
        if [ "${AE}" == "${EE}" ]; then
            seen=1
            echo "Expected env found: ${AE}"
            break
        fi
    done
done
if [ ${seen} -eq 0 ]; then
    echo "Error: not all expected envs were found"
    exit 1
fi

for EV in ${EXPECTED_VOLUMES}; do
    seen=0
    for AV in ${ALL_VOLUMES}; do
        if [ "${AV}" == "${EV}" ]; then
            echo "Expected volume found: ${AV}"
            break
        fi
    done
    for AVM in ${ALL_VOLUME_MOUNTS}; do
        if [ "${AVM}" == "${EV}" ]; then
            seen=1
            echo "Expected volume mount found: ${AVM}"
            break
        fi
    done
done
if [ ${seen} -eq 0 ]; then
    echo "Error: not all expected volumes and their mounts were found"
    exit 1
fi
