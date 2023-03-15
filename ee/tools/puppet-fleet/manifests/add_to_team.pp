define fleet::add_to_team () {
    $fleet_host = lookup('fleet::host', String)
    $fleet_token = lookup('fleet::token', String)

    $udid = $facts['system_profiler']['hardware_uuid']
    $out = add_host_to_team($udid, $name, $fleet_host, $fleet_token)
    $error = $out['error']
    if $error {
      notify{"Error adding host ${name} to team ${team}: ${error_message}": loglevel => 'err'}
    } else {
      notify{"Added host ${udid} to team ${name}": }
    }
}
