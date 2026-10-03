# qacheck

Scans a GitHub Project v2 for items in ✔️Awaiting QA
that are missing or have an unchecked QA confirmation checklist.

## Build

export GITHUB_TOKEN=...
go mod tidy
go build -o qacheck .

## Run

./qacheck -org fleetdm -project 71
./qacheck -org fleetdm -project 97