#!/bin/bash

setup_gcloud() {
    sudo /opt/google-cloud-sdk/bin/gcloud --quiet components update
    sudo /opt/google-cloud-sdk/bin/gcloud --quiet components update kubectl
    echo $GCLOUD_SERVICE_KEY | base64 --decode -i > ${HOME}/gcloud-service-key.json
    sudo /opt/google-cloud-sdk/bin/gcloud auth activate-service-account --key-file ${HOME}/gcloud-service-key.json
    sudo /opt/google-cloud-sdk/bin/gcloud config set project $PROJECT_NAME
    sudo /opt/google-cloud-sdk/bin/gcloud --quiet config set container/cluster $CLUSTER_NAME
    # Reading the zone from the env var is not working so we set it here
    sudo /opt/google-cloud-sdk/bin/gcloud config set compute/zone ${CLOUDSDK_COMPUTE_ZONE}
    sudo /opt/google-cloud-sdk/bin/gcloud --quiet container clusters get-credentials $CLUSTER_NAME

    sudo chown -R ubuntu:ubuntu /home/ubuntu/.kube
    sudo chown -R ubuntu:ubuntu /home/ubuntu/.config
}


setup_gcloud

# configure slack notification webhooks
echo $SLACKTEE_CONFIG | base64 --decode -i > ${HOME}/.slacktee
