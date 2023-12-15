#!/bin/bash

set -e

VERSION="0.1.0"
PLUGIN_PATH="${HOME}/.terraform.d/plugins/registry.terraform.io/paultyng/git/${VERSION}/darwin_arm64"
TERRAFORMRC="${HOME}/.terraformrc"

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

if [ -f "${TERRAFORMRC}" ]; then
	echo "Moving ${TERRAFORMRC} to ${TERRAFORMRC}.old before replacing"
	mv "${TERRAFORMRC}" "${TERRAFORMRC}.old"
fi
cat <<-EOF > "${HOME}/.terraformrc"
	provider_installation {
	  filesystem_mirror {
	    path    = "/Users/$(whoami)/.terraform.d/plugins"
	  }
	  direct {
	    exclude = ["registry.terraform.io/paultyng/*"]
	  }
	}
EOF
