# Enrollments

"Enrollments" hold settings for devices that will be enrolled to MDM.
The MDM "enrollments" will allow Fleet to automatically enroll devices to specific teams, which then allows for applying specific MDM settings (depending on the team).

For Dogfood-MVP, Fleet will allow creating global enrollments only (team support will be added at a subsequent iteration).
Users will be able to create the two following types of enrollments:
- Global manual enrollment
- Global DEP enrollment

We'll have a new `apple_enrollments` table with the following fields:
- `id` (used to deduce an "Enroll URL")
- `name`
- `config JSON`: holds enrollment config like "PayloadDisplayName", "AccessRights", TODO(lucas): Define with Guillaume if necessary.
- `dep_config JSON`: holds DEP enrollment profile (`NULL` when enroll is manual).
- `team` (`0` for global)

Fleetctl commands (and APIs):
- Create enrollments:
	`fleetctl apple-mdm enrollments create-automatic --name=Foo --config=<config.json> --profile=<dep_profile.json>`
	`fleetctl apple-mdm enrollments create-manual --name=Bar --config=<config.json>`

- List enrollments (the "global" manual enroll and the DEP enroll):
	`fleetctl apple-mdm enrollments list`

- Delete enrollments:
    `fleetctl apple-mdm enrollments delete --id=<ID> --config=<config.json>`