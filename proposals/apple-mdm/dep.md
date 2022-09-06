# DEP

## Commands

`fleetctl apple-mdm dep list`
Fetches a list of all devices that are assigned to the ABM's "MDM server" at the time of the request (via [get_a_list_of_devices](https://developer.apple.com/documentation/devicemanagement/get_a_list_of_devices)).

## DEP syncer

Fleet runs a "DEP syncer" routine to fetch newly added devices from ABM (via [get_a_list_of_devices](https://developer.apple.com/documentation/devicemanagement/get_a_list_of_devices)) and automatically apply the configured DEP profile to them (via [assign_a_profile](https://developer.apple.com/documentation/devicemanagement/assign_a_profile)).