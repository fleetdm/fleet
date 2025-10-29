# MacOS setup experience on a Virtual Machine

This is a quick and dirty tool that does some direct SQL queries to set up the necessary state, and therefore comes with some inherent brittleness. 

To use:

1. Start a local server [with MDM enabled](https://github.com/fleetdm/fleet/blob/main/docs/Contributing/getting-started/testing-and-local-development.md#mdm-setup-and-testing).
2. Ensure that end-user validation is _disabled_ in setup experience config.
3. Ensure no bootstrap package is uploaded.
4. Ensure no custom setup profile is uploaded.
5. Add some software and/or scripts to the setup experience config.
6. Enroll your macOS VM into a team. 
7. Get the UUID of the VM, either via a live query on Fleet (`SELECT uuid FROM osquery_info`), by inspecting the API response from the `/fleet/device/:token` endpoint on the My Device page, or querying the `hosts` table of the MySQL database directly.
7. Run this tool with the appropriate flags to set up the necessary database records, e.g.:

```bash
go run main.go -server-private-key=$(cat ~/path/to/private/key) -host-uuid="your-enrolled-host-uuid"
```

If the setup dialog doesn't appear on the VM, or it remains on the initial setup screen, try running the tool again and waiting.

Note that the setup experience dialog may not auto-dismiss after completing. You can dismiss manually it by pressing Command-Shift-X. To test the dialog again, run this tool again and restart Orbit on the device.
