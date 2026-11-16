#!/usr/bin/env bash
#
# Builds a multi-arch (amd64 + arm64) fleetdm/docker-idp image and pushes it under a new tag.
#
# The original image (pushed 2021-03-29, amd64 only) has no Dockerfile in any fleetdm repo. It is
# kristophjunge/docker-test-saml-idp with the changes below, recovered from the image's layer
# history. All baked-in certs/keys and config files are byte-identical to upstream except
# saml20-sp-remote.php (docker-compose.yml mounts tools/saml/config.php over it anyway).
#
# The script does not touch :latest. It also copies the current :latest to a backup tag so the
# original image stays reachable.
#
# Usage:
#   DOCKERHUB_USERNAME=... DOCKERHUB_PASSWORD=... ./build-push-docker-idp.sh
#
# DOCKERHUB_PASSWORD must be a Docker Hub Personal Access Token with Read & Write scope; Docker
# Hub rejects account passwords for accounts that require a PAT.
#
# Optional env vars:
#   TAG         Tag to push (default: simplesamlphp-1.19.0). Refuses to overwrite an existing tag.
#   BACKUP_TAG  Tag for the backup of the current :latest (default: legacy-amd64).
#   NO_PUSH=1   Build for the host architecture only and load it into the local Docker as
#               fleetdm/docker-idp:$TAG. No credentials needed; nothing is pushed.

set -euo pipefail

IMAGE="fleetdm/docker-idp"
TAG="${TAG:-simplesamlphp-1.19.0}"
BACKUP_TAG="${BACKUP_TAG:-legacy-amd64}"
NO_PUSH="${NO_PUSH:-}"

UPSTREAM_COMMIT="b15078a89d99216b3ed43221ab321e6eae4bd582"
UPSTREAM_TARBALL="https://github.com/kristophjunge/docker-test-saml-idp/archive/${UPSTREAM_COMMIT}.tar.gz"

if [ -z "$NO_PUSH" ]; then
  : "${DOCKERHUB_USERNAME:?DOCKERHUB_USERNAME must be set}"
  : "${DOCKERHUB_PASSWORD:?DOCKERHUB_PASSWORD must be set}"
fi

workdir="$(mktemp -d)"
builder="docker-idp-builder-$$"
cleanup() {
  docker buildx rm "$builder" >/dev/null 2>&1 || true
  rm -rf "$workdir"
}
trap cleanup EXIT

echo "==> Downloading kristophjunge/docker-test-saml-idp@${UPSTREAM_COMMIT}"
curl -fsSL "$UPSTREAM_TARBALL" | tar xz -C "$workdir" --strip-components=1

cat > "$workdir/Dockerfile" <<'EOF'
FROM php:7.4.16-apache
LABEL maintainer="Fleet Developers <engineering@fleetdm.com>"

# SimpleSAMLphp
ARG SIMPLESAMLPHP_VERSION=1.19.0
RUN curl -s -L -o /tmp/simplesamlphp.tar.gz https://github.com/simplesamlphp/simplesamlphp/releases/download/v$SIMPLESAMLPHP_VERSION/simplesamlphp-$SIMPLESAMLPHP_VERSION.tar.gz && \
    tar xzf /tmp/simplesamlphp.tar.gz -C /tmp && \
    mv /tmp/simplesamlphp-* /var/www/simplesamlphp && \
    rm -rf /tmp/* && \
    touch /var/www/simplesamlphp/modules/exampleauth/enable
COPY config/simplesamlphp/config.php /var/www/simplesamlphp/config
COPY config/simplesamlphp/authsources.php /var/www/simplesamlphp/config
COPY config/simplesamlphp/saml20-sp-remote.php /var/www/simplesamlphp/metadata
COPY config/simplesamlphp/server.crt /var/www/simplesamlphp/cert/
COPY config/simplesamlphp/server.pem /var/www/simplesamlphp/cert/

# Apache
COPY config/apache/ports.conf /etc/apache2
COPY config/apache/simplesamlphp.conf /etc/apache2/sites-available
COPY config/apache/cert.crt /etc/ssl/cert/cert.crt
COPY config/apache/private.key /etc/ssl/private/private.key
RUN chmod 655 /etc/ssl/private /etc/ssl/cert
RUN echo "ServerName localhost" >> /etc/apache2/apache2.conf && \
    a2enmod ssl && \
    a2dissite 000-default.conf default-ssl.conf && \
    a2ensite simplesamlphp.conf

# OpenSSL 1.1.1's ARM CPU feature probing crashes (SIGILL) on Apple M4 and newer, killing every
# HTTPS request. Disabling the probing is harmless; the variable is ignored on amd64.
ENV OPENSSL_armcap=0

WORKDIR /var/www/simplesamlphp
USER www-data
EXPOSE 8080 8443
EOF

cat > "$workdir/config/simplesamlphp/saml20-sp-remote.php" <<'EOF'
<?php
/**
 * SAML 2.0 remote SP metadata for SimpleSAMLphp.
 *
 * See: https://simplesamlphp.org/docs/stable/simplesamlphp-reference-sp-remote
 */

$metadata[getenv('SIMPLESAMLPHP_SP_ENTITY_ID')] = array(
    'AssertionConsumerService' => getenv('SIMPLESAMLPHP_SP_ASSERTION_CONSUMER_SERVICE'),
    'SingleLogoutService' => getenv('SIMPLESAMLPHP_SP_SINGLE_LOGOUT_SERVICE'),
    'NameIDFormat' => 'urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress',

    // refer to https://simplesamlphp.org/docs/stable/saml:nameid
    'authproc' => array(
        3 => array(
            'class' => 'saml:AttributeNameID',
            'attribute' => 'email',
            'Format' => 'urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress',
        ),
    ),
);
EOF

if [ -n "$NO_PUSH" ]; then
  echo "==> Building ${IMAGE}:${TAG} for the host architecture (NO_PUSH set, nothing will be pushed)"
  docker buildx build --load -t "${IMAGE}:${TAG}" "$workdir"
  exit 0
fi

echo "==> Logging in to Docker Hub as ${DOCKERHUB_USERNAME}"
printf '%s' "$DOCKERHUB_PASSWORD" | docker login --username "$DOCKERHUB_USERNAME" --password-stdin

if docker buildx imagetools inspect "${IMAGE}:${TAG}" >/dev/null 2>&1; then
  echo "error: ${IMAGE}:${TAG} already exists; set TAG to a new value" >&2
  exit 1
fi

if docker buildx imagetools inspect "${IMAGE}:${BACKUP_TAG}" >/dev/null 2>&1; then
  echo "==> Backup ${IMAGE}:${BACKUP_TAG} already exists, leaving it as is"
else
  echo "==> Backing up ${IMAGE}:latest to ${IMAGE}:${BACKUP_TAG}"
  docker buildx imagetools create -t "${IMAGE}:${BACKUP_TAG}" "${IMAGE}:latest"
fi

# The default "docker" buildx driver can't always build and push multi-platform images.
docker buildx create --name "$builder" --driver docker-container >/dev/null

echo "==> Building and pushing ${IMAGE}:${TAG} for linux/amd64,linux/arm64"
docker buildx build \
  --builder "$builder" \
  --platform linux/amd64,linux/arm64 \
  -t "${IMAGE}:${TAG}" \
  --push \
  "$workdir"

echo "==> Pushed:"
docker buildx imagetools inspect "${IMAGE}:${TAG}"
echo
echo "Next: point the saml_idp service in docker-compose.yml at ${IMAGE}:${TAG} in a PR."
