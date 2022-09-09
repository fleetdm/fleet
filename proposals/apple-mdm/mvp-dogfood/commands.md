# Commands

These are one-off commands like `"DeviceLock"`, `"RestartDevice"`, etc.

## Fleetctl commands

Send command:
`fleetctl apple-mdm commands send --target-hosts=1,2,3 --command=foo.plist`
fleetctl would be agnostic to contents of "foo.plist", such plist files can be generated with `./tools/mdm/apple/cmdr.py`

List commands (with their status):
`fleetctl apple-mdm commands list`