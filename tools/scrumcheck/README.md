# scrumcheck

Scans a GitHub Project v2 for items in ✔️Awaiting QA
that are missing or have an unchecked QA confirmation checklist.

## Build

export GITHUB_TOKEN=...
cd tools/scrumcheck
go build -o scrumcheck .

## Run

./scrumcheck -org fleetdm -project 71
./scrumcheck -org fleetdm -project 97
