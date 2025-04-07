Proxy for MDM requests used in seamless migrations, as described in
https://github.com/fleetdm/fleet/issues/19387.


### Usage

```
Usage of ./mdmproxy:
  -auth-token string
        Auth token for remote flag updates (remote updates disabled if not provided)
  -existing-hostname string
        Hostname for existing MDM server (eg. 'mdm.example.com') (required)
  -existing-url string
        Existing MDM server URL (full path) (required)
  -fleet-url string
        Fleet MDM server URL (full path) (required)
  -migrate-percentage int
        Percentage of clients to migrate from existing MDM to Fleet
  -migrate-udids string
        Space/newline-delimited list of UDIDs to migrate always
  -server-address string
        Address for server to listen on (default ":8080")
```

### Example invocation
```
mdmproxy --migrate-udids '' --auth-token foo --existing-url https://3.14.233.249 --existing-hostname micromdm.example.com --fleet-url https://example.cloud.fleetdm.com --migrate-percentage 0
```

### Check migration status

To check the migration status for a given UDID, provide the `--migrate-udids` and
`--migrate-percentage` flags with the `--check` flag:

```
$ go run . --migrate-percentage=50 --check E5C6DBBA-D5CC-4DB6-9560-995F17FB7A59
E5C6DBBA-D5CC-4DB6-9560-995F17FB7A59 IS NOT migrated
$ go run . --migrate-percentage=50 --check 575424CB-09D7-4CAD-8A7A-D3511FE8A7E2
575424CB-09D7-4CAD-8A7A-D3511FE8A7E2 IS migrated
```

When the `--check` flag is used, the program prints the migration status and exits. The server is not started.