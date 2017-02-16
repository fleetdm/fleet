#!/bin/bash

fpm -s dir -t deb --deb-no-default-config-files -n "kolide" -v ${KOLIDE_VERSION} /pkgroot/usr/=/usr
fpm -s dir -t rpm -n "kolide" -v ${KOLIDE_VERSION} /pkgroot/usr/=/usr
mv kolide* /out

# sign packages
rpmVersion="$(echo ${KOLIDE_VERSION}|sed 's/-/_/g')"
rpm --addsign "/out/kolide-${rpmVersion}-1.x86_64.rpm"
debsigs --sign=origin -k 000CF27C "/out/kolide_${KOLIDE_VERSION}_amd64.deb"
