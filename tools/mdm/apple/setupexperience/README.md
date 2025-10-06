# MacOS setup experience on a Virtual Machine

This is a quick and dirty tool that does some direct SQL queries to set up the necessary state, and therefore comes with some inherent brittleness. 

To use:

1. Start a local server [with MDM enabled](https://github.com/fleetdm/fleet/blob/main/docs/Contributing/getting-started/testing-and-local-development.md#mdm-setup-and-testing).
2. Ensure that end-user validation is _disabled_ in setup experience config.
3. Ensure no bootstrap package is uploaded.
4. Ensure no custom setup profile is uploaded.
5. Add some software and/or scripts to the setup experience config.
6. Enroll your macOS VM into a team. 
7. Run this tool with the appropriate flags to set up the necessary database records, e.g.:

```
go run main.go -server-private-key=$(cat ~/path/to/private/key) -host-uuid="your-enrolled-host-uuid"
```

If the setup dialog doesn't appear on the VM, or it remains on the initial setup screen, try running the tool again and waiting.

Note that unless your instance is configured to enable manually releasing devices, the setup experience dialog will not auto-dismiss after completing.
