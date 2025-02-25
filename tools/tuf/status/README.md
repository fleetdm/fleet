# TUF status

The TUF status tool can be used to process information of a Fleet TUF repository hosted on AWS S3 or Cloudflare R2.
The default URL is Fleet's TUF: https://updates.fleetctl.com.

# Get the version numbers of a channel

To get the version numbers of components in a given channel you can use the `channel-version` command.
```sh
go run tools/tuf/status/tuf-status.go channel-version -channel stable
{
  "desktop": {
    "linux": "1.20.0",
    "macos": "1.20.0",
    "windows": "1.20.0"
  },
  "nudge": {
    "macos": "1.1.10.81462"
  },
  "orbit": {
    "linux": "1.20.1",
    "macos": "1.20.1",
    "windows": "1.20.1"
  },
  "osqueryd": {
    "linux": "5.9.1",
    "macos": "5.9.1",
    "windows": "5.9.1"
  },
  "swiftDialog": {
    "macos": "2.1.0"
  }
}
```