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

if [ "$INFRA_NAME" == "all" ]; then
    INFRAS=("mysql" "redis" "mailhog" "saml_idp")
else
    INFRAS=("$INFRA_NAME")
fi

for INFRA in ${INFRAS[@]}; do
    echo "check whether the $INFRA is ready"
    if [ "$INFRA" == "mysql" ]; then
        docker-compose exec -T mysql_test bash -c 'echo "SHOW DATABASES;" | mysql -uroot -ptoor'
        echo "mysql is ready!"
    elif [ "$INFRA" == "redis" ]; then
        docker-compose exec -T redis bash -c "redis-cli ping"
        echo "redis is ready!"
    elif [ "$INFRA" == "mailhog" ]; then
        date | mail -s "Test email" recipient@test.com
        echo "mailhog is ready!"
    elif [ "$INFRA" == "saml_idp" ]; then
        echo "TODO"
        echo "saml_idp is ready!"
    fi
done
