# @summary Add a configuration profile to the device.
#
# This resource ensures that the provided configuration
# profile will be applied to the device using Fleet.
#
# Fleet keeps track of all the times this resource is
# called for a device during a Puppet sync and at the
# end of the sync tries to match the set of profiles
# to an existing team and assign the device to the team.
#
# If a team doesn't exist, Fleet will automatically create one.
#
# @param template
#   XML with the profile definition.
# @param group
#   Used to define the team name in Fleet.
#   Fleet keeps track of each time this resource is
#   declared with a group name, the final team name
#   will be a concatenation of all unique group names.
#
# @example
#   fleetdm::profile { 'identifier': }
define fleetdm::profile (
  String $template,
  String $group = 'default',
) {
  if $facts["clientnoop"] {
    notice('noop mode: skipping profile definition in the Fleet server')
  }  else {
    unless $template =~ /^[[:print:]]+$/ {
      fail('invalid template')
    }

    unless $group =~ /^[[:print:]]+$/ {
      fail('invalid group')
    }

    $host_uuid = $facts['system_profiler']['hardware_uuid']
    $response = fleetdm::preassign_profile($name, $host_uuid, $template, $group)
    $err = $response['error']

    if $err != '' {
      notify { "error pre-assigning profile ${$name}: ${$err}":
        loglevel => 'err',
      }
    } else {
      notify { "successfully pre-assigned profile ${$name}": }
    }
  }
}
