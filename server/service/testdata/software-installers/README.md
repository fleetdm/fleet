# testdata

- `fleet-osquery.msi` is a dummy MSI installer created by `packaging.BuildMSI` with a fake `orbit.exe` that just has `hello world` in it. Its software title is `Fleet osquery` and its version is `1.0.0`.
- `ruby.rpm` was downloaded from https://rpmfind.net/linux/fedora/linux/development/rawhide/Everything/x86_64/os/Packages/r/ruby-3.3.5-15.fc42.x86_64.rpm.
- `no_bundle_identifier.pkg` was generated with the following command `pkgbuild --nopayload --install-location "/" --scripts scripts/ --identifier ' ' --version '1.0.0' no_bundle_identifier.pkg` (where `scripts/` contained a dummy `preinstall` and `postinstall` scripts).