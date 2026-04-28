#!/bin/bash

# Please don't delete. This script is linked to, as a redirect, from fleetctl and
# the Fleet website. It is preserved as an informational endpoint at
# https://fleetdm.com/install-wine so existing links don't 404.

cat <<'EOF'

============================================================
This script no longer installs Wine.
============================================================

Wine is no longer required to build Windows (.msi) packages on macOS.
fleetctl package now uses Docker by default on all macOS architectures.

RECOMMENDED: install Docker Desktop
  https://docs.docker.com/get-docker

If you cannot use Docker and still need to build MSIs with Wine on macOS
see the upstream WineHQ wiki for installation instructions:
  https://gitlab.winehq.org/wine/wine/-/wikis/MacOS

Automatic Wine installation via Homebrew is no longer attempted here
because the wine-stable cask is deprecated and upstream Wine releases
have caused repeated breakage.

EOF

exit 1
