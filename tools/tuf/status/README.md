# TUF status

The TUF status tool can be used to process information of a Fleet TUF repository hosted on AWS S3.
The default URL is Fleet's TUF: https://tuf.fleetctl.com.

# Fetch and filter targets

To get information of targets you can use the `key-filter` command.

E.g. to get all targets filtering by the `edge` channel:
```sh
go run tools/tuf/status/tuf-status.go key-filter -filter edge

Results filtered by "edge" and sorted by version, platform and key.

VERSION PLATFORM        KEY                                                     LAST MODIFIED                   SIZE    ETAG
edge    linux           targets/desktop/linux/edge/desktop.tar.gz               2024-01-09T20:51:49.000Z        16.3 MB "da05e73b8b351299f1d7063afb538529-2"
edge    linux           targets/orbit/linux/edge/orbit                          2024-01-19T21:35:09.000Z        40.7 MB "a38ff2a2e47b73fe1456563126a8db6d-5"
edge    linux           targets/osqueryd/linux/edge/osqueryd                    2024-01-03T22:19:35.000Z        86.5 MB "8d7e48d9e9883013bfc493d44b96b4e7-11"
edge    macos           targets/desktop/macos/edge/desktop.app.tar.gz           2024-01-09T20:52:04.000Z        31.9 MB "37e3048387d1f2724fb90417126f8444-4"
edge    macos           targets/orbit/macos/edge/orbit                          2024-01-19T21:36:58.000Z        83.9 MB "7d9bc91b9ce6b5234195650c082d9b9b-11"
edge    macos-app       targets/osqueryd/macos-app/edge/osqueryd.app.tar.gz     2024-01-03T22:19:47.000Z        24.4 MB "653a3f86b2607798592de3c73a88b1f0-3"
edge    windows         targets/desktop/windows/edge/fleet-desktop.exe          2024-01-09T20:52:13.000Z        36.8 MB "a482a0e4f0b57e89e6846bd65b8d8ab1-5"
edge    windows         targets/orbit/windows/edge/orbit.exe                    2024-01-19T21:38:37.000Z        40.7 MB "b90014b53abf013fc1bdaec39ab03683-5"
edge    windows         targets/osqueryd/windows/edge/osqueryd.exe              2024-01-03T22:19:52.000Z        24.8 MB "2887ba627688255d9ec009fbe7b02fbf-3"
```

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