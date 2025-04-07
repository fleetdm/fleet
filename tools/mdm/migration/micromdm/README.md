# MicroMDM webhook

A tiny server you can use as a webhook callback for the MDM migration [end user workflow](https://fleetdm.com/docs/using-fleet/mdm-migration-guide#end-user-workflow).

It will try to unenroll the device based on the device UUID/UDID by sending a `RemoveProfile`
command.

## Usage

1. Find the MicroMDM API token. For the Fly.io hosted MicroMDM server it should be in
   1Password. If you're having trouble finding it, drop a message in `#g-mdm` on Slack!
2. Get the MicroMDM server URL.
3. Start the server with:

```
go run tools/mdm/migration/micromdm/main.go --api-token=$MICRO_MDM_TOKEN --url=https://micromdm.example.com
```

4. Configure Fleet to send a webhook to this server.