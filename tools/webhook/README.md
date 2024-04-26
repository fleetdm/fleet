# webhook

Test tool for Fleet features that use webhook URLs.
It reads and parses the request a JSON body and prints the JSON to standard output (with indentation).

```sh
go run ./tools/webhook 8082
2024/03/20 09:10:00 {
  "error": "No fleetdm.com Google account associated with this host.",
  "host_display_name": "dChYnk.uxURT",
  "host_id": 2,
  "host_serial_number": "",
  "timestamp": "2024-03-20T09:10:00.129982-03:00"
}
...
```
