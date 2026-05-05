### SimpleMDM Webhook

A tiny web server you can use as a webhook callback for the MDM migration [end user
workflow](https://fleetdm.com/docs/using-fleet/mdm-migration-guide#end-user-workflow) for a given
device from SimpleMDM to Fleet.

Use the `api-token` flag to specify the API token for your SimpleMDM server. Use the `device-id`
flag to specify the device ID you want to migrate. The device ID is the numerical ID of the device
in SimpleMDM and can be found in the URL for the device details page in SimpleMDM.

This is useful for testing and local development.

#### Usage

1. Start the webserver with:

```
go run tools/mdm/migration/simplemdm/main.go --api-token=<YOUR_API_TOKEN> --device-id=<YOUR_DEVICE_ID>
```

Output will be printed to stdout.

```
Server running at http://localhost:4648
```

2. Configure an https proxy (e.g., ngrok) to forward requests to localhost.

3. Use the https address when configuring the Fleet migration webhook URL.
