module github.com/fleetdm/fleet/v4/tools/mdm/migration/micromdm/touchless

go 1.27.4

// Kept as its own module so the MicroMDM platform packages it reads (and their gorilla/mux, buford and boltdb
// dependencies) stay out of the main fleet module. The parent module is pinned to the local checkout for the SCEP depot.
replace github.com/fleetdm/fleet/v4 => ../../../../..

require (
	github.com/boltdb/bolt v1.3.1
	github.com/fleetdm/fleet/v4 v4.0.0-00010101000000-000000000000
	github.com/micromdm/micromdm v1.9.0
	github.com/micromdm/plist v0.4.0
	go.etcd.io/bbolt v1.4.0
)

require (
	github.com/RobotsAndPencils/buford v0.14.0 // indirect
	github.com/garyburd/go-oauth v0.0.0-20180319155456-bca2e7f09a17 // indirect
	github.com/go-kit/kit v0.13.0 // indirect
	github.com/go-kit/log v0.2.1 // indirect
	github.com/go-logfmt/logfmt v0.5.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/mux v1.8.1 // indirect
	github.com/groob/plist v0.0.0-20220217120414-63fa881b19a5 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/smallstep/pkcs7 v0.2.3 // indirect
	github.com/smallstep/scep v0.0.0-20241223071629-a37a330173bc // indirect
	go.mozilla.org/pkcs7 v0.9.0 // indirect
	golang.org/x/net v0.62.0 // indirect
	golang.org/x/sys v0.50.0 // indirect
	golang.org/x/text v0.43.0 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)
