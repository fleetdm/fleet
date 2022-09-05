# Commands

These are one-off commands like "DeviceLock", "RestartDevice", etc.

Fleetctl commands:
- Send command:
	`fleetctl apple-mdm commands send --target-hosts=1,2,3 --command=foo.plist`
(fleetctl would be agnostic to contents of "foo.plist")
- List commands (with their status):
	`fleetctl apple-mdm commands list`