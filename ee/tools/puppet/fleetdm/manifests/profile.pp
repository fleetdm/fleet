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
# @param ensure
#   Whether the profile should be present or not.
#   Set to `absent` along with a distinct `group`
#   name to create a new team that doesn't have the
#   configuration profile. 
#
# @example
#   fleetdm::profile { 'identifier': }
define fleetdm::profile (
  String $template,
  String $group = 'default',
  Enum['absent', 'present'] $ensure = 'present',
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
    $response = fleetdm::preassign_profile($name, $host_uuid, $template, $group, $ensure)
    $err = $response['error']
    $changed = $response['resource_changed']

    if $err != '' {
      notify { "error pre-setting fleetdm::profile ${name} as ${ensure}: ${err}":
        loglevel => 'err',
      }
    } elsif $changed {
      # NOTE: sending a notification also marks the
      # 'fleetdm::profile' as changed in the reports.
      notify { "successfully pre-set fleetdm::profile ${name} as ${ensure}": }
    }
  }
}
