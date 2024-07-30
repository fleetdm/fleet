# Copyright (c) 2020-present, The kubequery authors
#
# This source code is licensed as defined by the LICENSE file found in the
# root directory of this source tree.
#
# SPDX-License-Identifier: (Apache-2.0 OR GPL-2.0-only)

FROM ubuntu:20.04@sha256:80ef4a44043dec4490506e6cc4289eeda2d106a70148b74b5ae91ee670e9c35d AS builder

ARG BASEQUERY_VERSION=5.0.2

ADD https://uptycs-basequery.s3.amazonaws.com/${BASEQUERY_VERSION}/basequery_${BASEQUERY_VERSION}-1.linux_amd64.deb /tmp/basequery.deb

RUN dpkg -i /tmp/basequery.deb

# =====

FROM uptycs/busybox:v1.33.0@sha256:6a312f5959d374420eedce83f42d2ad19a027bd4e448ed734372bc1a07ad8b10

ARG BASEQUERY_VERSION
ARG KUBEQUERY_VERSION

LABEL \
  name="kubequery" \
  description="kubequery powered by Osquery" \
  version="${KUBEQUERY_VERSION}" \
  url="https://github.com/Uptycs/kubequery"

# uptycs/busybox comes with this user predefined. We need a non-root user
USER uptycs

WORKDIR /opt/uptycs

RUN set -ex; \
    mkdir /opt/uptycs/bin /opt/uptycs/etc /opt/uptycs/logs /opt/uptycs/var && \
    echo "/opt/uptycs/bin/kubequery.ext" > /opt/uptycs/etc/autoload.exts

COPY --from=0 --chown=uptycs:uptycs /opt/osquery/bin/osqueryd /opt/uptycs/bin/basequery
COPY --from=0 --chown=uptycs:uptycs /opt/osquery/share/osquery/certs/certs.pem /opt/uptycs/etc/
COPY --chown=uptycs:uptycs bin/entrypoint.sh bin/kubequeryi bin/uuidgen /opt/uptycs/bin/
COPY --chown=uptycs:uptycs bin/kubequery /opt/uptycs/bin/kubequery.ext

ENV KUBEQUERY_VERSION=${KUBEQUERY_VERSION}

ENTRYPOINT ["/opt/uptycs/bin/entrypoint.sh"]
