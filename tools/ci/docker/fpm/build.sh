#!/bin/bash

fpm -s dir -t deb --deb-no-default-config-files -n "kolide" -v ${KOLIDE_VERSION} /pkgroot/usr/=/usr
fpm -s dir -t rpm -n "kolide" -v ${KOLIDE_VERSION} /pkgroot/usr/=/usr
mv kolide* /out
