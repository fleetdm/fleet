#!/bin/bash

set -e

VERSION="0.1.0"
PLUGIN_PATH="${HOME}/.terraform.d/plugins/registry.terraform.io/paultyng/git/${VERSION}/darwin_arm64"

pushd /tmp
git clone git@github.com:paultyng/terraform-provider-git.git
cd terraform-provider-git
# Tag format is v0.1.0 but folder above has no 'v'
git checkout v${VERSION}
go build -o terraform-provider-git .
mkdir -p "${PLUGIN_PATH}"
mv terraform-provider-git ${PLUGIN_PATH}
cd ..
rm -rf /tmp/terraform-provider-git
popd
