#!/bin/bash

mkdir -p ~/deps

EXEC_TEMPLATE="https://github.com/groob/exec-template/releases/download/1.0.0/exec-template-linux-amd64"
SLACKTEE="https://raw.githubusercontent.com/course-hero/slacktee/77ea95faa12b1b6114e32f336440f4ea4a249e99/slacktee.sh"

docker_pull() {
    local_image="/home/ubuntu/deps/$1.tar"
    remote_image=$2

    if [[ -e $local_image ]]; then
        docker load -i $local_image;
    else
        docker pull "${remote_image}"
        docker save "${remote_image}" > "${local_image}"
    fi
}

curl_binary() {
    url=$1
    out=$2

    if [ ! -f "${out}" ]; then
        curl -L "${url}" -o "${out}"
        chmod a+x "${out}"
    fi
}

docker_pull "kolide_builder" "kolide/kolide-builder:1.8-yarn"
docker_pull "redis" "redis"
docker_pull "mysql" "mysql:5.7"
docker_pull "fpm" "kolide/fpm"
docker_pull "cloudsql_proxy" "b.gcr.io/cloudsql-docker/gce-proxy:1.05"

curl_binary "${EXEC_TEMPLATE}" "/home/ubuntu/deps/exec-template"
curl_binary "${SLACKTEE}" "/home/ubuntu/deps/slacktee"
