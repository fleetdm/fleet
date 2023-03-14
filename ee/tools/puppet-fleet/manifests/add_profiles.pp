define fleet::add_profiles ($profiles) {
    $fleet_host = lookup('fleet::host', String)
    $fleet_token = lookup('fleet::token', String)

    $out = batch_send_profiles($name, $profiles, $fleet_host, $fleet_token)
    $error = $out['error']
    if $error {
      notify{"Error pushing profiles for team ${name}: ${error_message}": loglevel => 'err'}
    } else {
      notify{"Team ${name} profiles updated": }
    }
}
