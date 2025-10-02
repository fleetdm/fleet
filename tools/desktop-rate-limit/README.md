# desktop-rate-limit

Tool to brute force Fleet Desktop endpoints to test rate-limiting via IP banning.

Usage:
```sh
# -fleet_desktop_token can be set to a valid Fleet Desktop token to simulate good traffic alongside
# unauthenticated requests.
go run ./tools/desktop-rate-limit/main.go \
    -fleet_url https://localhost:8080 \
    -fleet_desktop_token 62381ddb-63a0-41da-bd2f-dd13a462cc6
```
