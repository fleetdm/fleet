#!/usr/bin/env bash

set -eou pipefail

usage() {
    echo "${0} <INFRA_NAME> 'CHECK WHETHER THE SPECIFIED INFRA DEPENDENCE IS READY"
}

if [ "$#" -ne 1 ] || [ -z "$1" ]; then
    echo "Error: Missing the infra name which needs to check"
    usage
    exit 1
fi


# infra is one of 'mysql', 'redis', 'mailhog', 'saml_idp'.
# use 'all' to check all these infras.
INFRA_NAME=${1}
INFRAS=()
RETRYNUM=10

if [ "$INFRA_NAME" == "all" ]; then
    INFRAS=("mysql" "redis" "mailhog" "saml_idp")
else
    INFRAS=("$INFRA_NAME")
fi

checkInfraFun() {
    INFRA=$1
    echo "check whether the $INFRA is ready"
    if [ "$INFRA" == "mysql" ]; then
        ! docker-compose exec -T database_test bash -c 'echo "SHOW DATABASES;" | mysql -uroot -ptoor' && return 1
        echo "mysql is ready!"
    elif [ "$INFRA" == "redis" ]; then
        ! docker-compose exec -T redis bash -c "redis-cli ping" && return 1
        echo "redis is ready!"
    elif [ "$INFRA" == "mailhog" ]; then
        echo "TODO"
        echo "mailhog is ready!"
    elif [ "$INFRA" == "saml_idp" ]; then
        echo "TODO"
        echo "saml_idp is ready!"
    fi
}

for INFRA in ${INFRAS[@]}; do
    n=0
    success=false
    until [ "$n" -ge $RETRYNUM ]; do
        checkInfraFun $INFRA && success=true && break
        n=$((n+1))
        sleep 1 
    done

    if [ ! $success ]; then
        exit 1
    fi
done
