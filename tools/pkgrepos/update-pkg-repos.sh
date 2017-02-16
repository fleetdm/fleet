#!/bin/bash

USER="$(whoami)"
LOCAL_REPO_PATH="/Users/${USER}/kolide_packages"
GPG_PATH="/Users/${USER}/.gnupg"

build_createrepo_container() {
    cd ../ci/docker/createrepo && \
        docker build -t createrepo . && cd -
}

build_aptly_container() {
    cd ../ci/docker/aptly && \
        docker build -t aptly . && cd -
}

update_yum_repo() {
    # generate new yum repo snapshot
    docker run -it --rm \
        -v "${LOCAL_REPO_PATH}/yum:/repo" \
        -v "${GPG_PATH}:/root/.gnupg" \
        createrepo
}

update_apt_repo() {
    docker run -it --rm \
        -v "${LOCAL_REPO_PATH}/deb:/deb" \
        -v "${GPG_PATH}:/root/.gnupg" \
        -v "${LOCAL_REPO_PATH}/aptly:/root/.aptly" \
        -v "${LOCAL_REPO_PATH}/aptly.conf:/root/.aptly.conf" aptly

    # replace "debian" repo with updated snapshot
    rm -rf "${LOCAL_REPO_PATH}/apt" 
    mv "${LOCAL_REPO_PATH}/aptly/public" "${LOCAL_REPO_PATH}/apt" 
}


main() {
    build_createrepo_container
    build_aptly_container
    update_yum_repo
    update_apt_repo
}

main
