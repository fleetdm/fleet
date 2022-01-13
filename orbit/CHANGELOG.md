## Orbit 0.0.5 (Dec 22, 2021)

* Fix handling of enroll secrets to address 0.0.4 enrollment issue.

## Orbit 0.0.4 (Dec 19, 2021)

* Use `certs.pem` if available in root directory to improve TLS compatibility.

* Use UUID as the default host identifier for osquery.

* Add github.com/macadmins/osquery-extension tables.

* Add support for osquery flagfile (loaded automatically if it exists in the Orbit root).

* Fix permissions for building MSI when packaging as root user. Fixes fleetdm/fleet#1424.

