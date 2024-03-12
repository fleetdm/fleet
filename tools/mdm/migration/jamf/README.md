# Jamf Pro Webhook

A tiny web server you can use as a webhook callback for the MDM migration [end user workflow](https://fleetdm.com/docs/using-fleet/mdm-migration-guide#end-user-workflow)

This will try to find a device using the serial number and send an API call to unenroll it.

#### Usage

1. Grab an [user and password](https://learn.jamf.com/en-US/bundle/jamf-pro-documentation-current/page/Jamf_Pro_User_Accounts_and_Groups.html). Make sure it has access rights to list and unenroll devices.
2. Grab your URL, for example if you're using Jamf Cloud `https://foo.jamfcloud.com`
3. Start the webserver with:

```
go run tools/mdm/migration/jamf/main.go --url=https://foo.jamfcloud.com --username=$JAMF_USERNAME --password=$JAMF_PASSWORD
```

4. Configure Fleet to send a webhook to your web server.

### TODO

- [ ] For long running servers in production environments, check the token expiration and renew when appropriately.
