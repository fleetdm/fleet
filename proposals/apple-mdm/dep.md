# DEP

## Commands

- List DEP devices: DEP devices of that enrollment and their status (using the "DEP proxy API").
	`fleetctl apple-mdm dep list`
- Sync DEP profiles: makes sure to set the enroll profile config for new devices in a DEP enrollment. (Fleet would still sync all DEP enrollments automatically every 5m.)
	`fleetctl apple-mdm dep sync-profiles`

## DEP syncer

Fleet runs a "DEP syncer" routine to fetch newly added devices to business.apple.com and automatically apply DEP enroll configuration to them.