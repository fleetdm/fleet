#!/bin/bash

REVSHORT="$(git rev-parse --short HEAD)"
exec_template="/home/ubuntu/deps/exec-template"
slacktee="/home/ubuntu/deps/slacktee"

start_cloudsql_proxy() {

    # remove in case already running
    docker rm -f cloudsql-proxy

    # run GCP cloudsql proxy container
    docker run -d -p 3310:3306 \
		-v /etc/ssl/certs:/etc/ssl/certs \
		--name cloudsql-proxy \
		-v /home/ubuntu/gcloud-service-key.json:/secrets/credentials.json \
		b.gcr.io/cloudsql-docker/gce-proxy:1.05 /cloud_sql_proxy \
			--dir=/cloudsql \
			-instances=kolide-ose-testing:us-east1:kolidepr01=tcp:0.0.0.0:3306 \
			-credential_file=/secrets/credentials.json

    # wait for mysql connection to be established
    sleep 5
}

# clones kolide_master into kolide_prNum_revshort
copy_db() {
    dbname=$1

    # create new DB for this PR
    echo "CREATE DATABASE IF NOT EXISTS ${dbname}" | \
        mysql -p${CLOUDSQL_PASS} \
        --user=${CLOUDSQL_USER} \
        -h 127.0.0.1 \
        --port=3310

    # clone db
    mysqldump \
        -p${CLOUDSQL_PASS} \
        --user=${CLOUDSQL_USER} \
        -h 127.0.0.1 \
        --port=3310 kolide_master| \
        mysql -p${CLOUDSQL_PASS} \
        --user=${CLOUDSQL_USER} \
        -h 127.0.0.1 \
        --port=3310 $dbname
}

migrate_kolide_db() {
    dbname=$1

    ./build/kolide prepare db --no-prompt \
        --mysql_address=127.0.0.1:3310 \
        --mysql_database=${dbname} \
        --mysql_username=${CLOUDSQL_USER} \
        --mysql_password=${CLOUDSQL_PASS}
}

deploy_pr() {
    jsn="{ \"Number\" : \"${CIRCLE_PR_NUMBER}\", \"RevShort\" : \"${REVSHORT}\"}"

    $exec_template -json="$jsn" -template=./tools/ci/k8s-templates/pr-service.template > /tmp/service.yml
    $exec_template -json="$jsn" -template=./tools/ci/k8s-templates/pr-deployment.template > /tmp/deployment.yml

    # TODO(@groob):
    # we have to deploy a new copy of redis for each PR. In the future,
    # it would be nice to deploy a single redis instance and allow multiple DBs to connect.
    $exec_template -json="$jsn" -template=./tools/ci/k8s-templates/redis-pr-service.template > /tmp/redis-service.yml
    $exec_template -json="$jsn" -template=./tools/ci/k8s-templates/redis-pr-deployment.template > /tmp/redis-deployment.yml

    kubectl apply -f /tmp/service.yml
    kubectl apply -f /tmp/deployment.yml

    kubectl apply -f /tmp/redis-service.yml
    kubectl apply -f /tmp/redis-deployment.yml

    echo "Deployed PR ${CIRCLE_PR_NUMBER}, commit ${CIRCLE_SHA1}" | \
        $slacktee \
        -c engineering \
        --title "${CIRCLE_PR_NUMBER}.kolide.kolide.net"  \
        --link "https://${CIRCLE_PR_NUMBER}.kolide.kolide.net" \
        -m full \
        -a good \
        -p
}

deploy_branch() {
    branch="${1}"
    jsn="{ \"Name\" : \"${branch}\", \"RevShort\" : \"${REVSHORT}\"}"

    $exec_template -json="$jsn" -template=./tools/ci/k8s-templates/branch-service.template > /tmp/service.yml
    $exec_template -json="$jsn" -template=./tools/ci/k8s-templates/branch-deployment.template > /tmp/deployment.yml

    kubectl apply -f /tmp/deployment.yml
    kubectl apply -f /tmp/service.yml

    echo "Deployed Branch ${branch}, commit ${CIRCLE_SHA1}" | \
        $slacktee \
        -c engineering \
        --title "${branch}.kolide.kolide.net"  \
        --link "https://${branch}.kolide.kolide.net" \
        -m full \
        -a good \
        -p
}

main() {
    start_cloudsql_proxy

    if [ -z ${CIRCLE_PR_NUMBER} ]; then
        dbname="kolide_master"
        migrate_kolide_db "${dbname}"
        deploy_branch "master"
    else
        dbname="pr_${CIRCLE_PR_NUMBER}_${REVSHORT}"
        copy_db "${dbname}"
        migrate_kolide_db "${dbname}"
        deploy_pr
    fi

    docker stop $(docker ps -a -q)
}

main
