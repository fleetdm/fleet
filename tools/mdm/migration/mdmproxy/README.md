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