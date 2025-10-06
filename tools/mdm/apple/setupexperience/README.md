# MacOS setup experience on a Virtual Machine

This is a quick and dirty tool that does some direct SQL queries to set up the necessary state, and therefore comes with some inherent brittleness. 

To use, first start a local server with MDM enabled, and enroll your macOS VM into a team. Then, run this tool with the appropriate flags to set up the necessary database records, e.g.:

```
go run main.go -server-private-key=$(cat ~/path/to/private/key) -host-uuid="your-enrolled-host-uuid"
```

If the setup dialog doesn't appear on the VM, or it remains on the initial setup screen, try running the tool again and waiting.

Note that unless your instance is configured to enable manually releasing devices, the setup experience dialog will not auto-dismiss after completing.
