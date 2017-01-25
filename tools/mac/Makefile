PKGNAME=kolide-enroll
PKGVERSION=1.0.0
PKGID=co.kolide.osquery.enroll

-include config.mk

export KOLIDE_FLAGS
export KOLIDE_TLS_CERTIFICATE


all: clean build
clean:
	rm -rf out/
	rm -rf root/etc/osquery

build: clean
	mkdir -p out
	mkdir -p root/etc/osquery
	echo $(ENROLL_SECRET) > root/etc/osquery/kolide_secret
	echo "$$KOLIDE_TLS_CERTIFICATE" > root/etc/osquery/kolide.crt
	
	# validate the certificate
	openssl x509 -in root/etc/osquery/kolide.crt -text > /dev/null
	echo "$$KOLIDE_FLAGS" > root/etc/osquery/kolide.flags
	pkgbuild --root root --scripts scripts --identifier ${PKGID} --version ${PKGVERSION} out/${PKGNAME}-${PKGVERSION}.pkg
