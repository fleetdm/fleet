# Profiles

A profile is defined as:
- `id`
- `uuid VARCHAR(255)`
- `name VARCHAR(255)`
- `payload BLOB` (raw XML plist)
- `team` (NULL for global)

For MVP-dogfood, we will only have global profiles, and as such, they are applied to all MDM enrolled hosts.

Fleetctl commands (and APIs):
- Create global profiles:
	`fleetctl apple-mdm profiles create --name="Chrome_Enrollment" --payload=foo.xml`
- List profile(s) (displays ID/UUID):
	`fleetctl apple-mdm profiles list`
- Delete profiles:
	`fleetctl apple-mdm profiles delete`
  TODO(Lucas): It will delete the profile from all hosts.

## TODO

- We have a list of all the profiles to apply to devices. How does Fleet know which profiles are
already deployed on a device and which aren't? (Are there any APIs to poll that work at scale?)
- How does Fleet determine if a profile needs to be updated or deleted on a device.
