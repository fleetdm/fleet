# Profiles

A profile is defined as:
- ID
- UUID
- Name
- Payload (raw XML plist)
- Team (NULL for global)

For MVP-dogfood, we will only have global profiles, and as such, they are applied to all MDM enrolled hosts.

TODO(Lucas): Stuff to solve around this feature:
So, we have a list of all the profiles to apply to devices. How does Fleet know which profiles are
already deployed on a device and which aren't? (Are there any APIs to poll that work at scale?)

Fleetctl commands (and APIs) to:
- Create global profiles:
	`fleetctl apple-mdm profiles create --name="Chrome_Enrollment" --payload=foo.xml`
- List profile(s) (displays ID/UUID):
	`fleetctl apple-mdm profiles list`
- Delete profiles:
	`fleetctl apple-mdm profiles delete`
  TODO(Lucas): It will delete the profile from all hosts.