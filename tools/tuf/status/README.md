# TUF status

```
~/fleet $ go run tools/tuf/status/status.go --help

This is a CLI utility to fetch and filter the entries posted by a TUF repository.

  -key-filter string
    	filter keys using a regular expression (default "stable")
  -url string
    	URL of the TUF repository (default "https://tuf.fleetctl.com")


Examples

- To filter all items on the edge channel use --key-filter="edge"
- To filter all items on version 1.3 including patches that run on Linux use --key-filter="linux/1.3.*"
- To filter Fleet Desktop items on 1.3.*, stable and edge that run on macOS use --key-filter="desktop/*.*/macos/(1.3.*|stable|edge)"
```
