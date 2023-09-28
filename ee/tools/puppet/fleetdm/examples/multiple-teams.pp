node default {
  fleetdm::profile { 'com.apple.SoftwareUpdate':
    template => template('fleetdm/automatic_updates.mobileconfig.erb'),
    group    => 'base',
  }

  fleetdm::profile { 'cis.macOSBenchmark.section2.BluetoothSharing':
    template => template('fleetdm/disable_bluetooth_file_sharing.mobileconfig.erb'),
    group    => 'workstations',
  }
}
