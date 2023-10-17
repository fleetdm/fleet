### Webhook

A tiny web server you can use as a webhook callback for the MDM migration [end user workflow](https://fleetdm.com/docs/using-fleet/mdm-migration-guide#end-user-workflow)

This will try to find a device using the serial number and send an API call to unenroll it.

#### Usage

1. Grab an API token from Kandji. Make sure it has access rights to list and unenroll devices.
2. Grab your subdomain, for example if your URL is `https://foo.kandji.com`, then grab `foo`.
3. Start the webserver with:

```
go run tools/mdm/migration/kandji/main.go --subdomain=foo --api-token=ABC-DEF
```

4. Configure Fleet to send a webhook to your web server.
