node default {
  fleetdm::profile { 'com.apple.universalaccess':
    template => 'xml template',
    group    => 'workstations',
  }

  fleetdm::profile { 'com.apple.homescreenlayout':
    template => 'xml template',
  }
}
