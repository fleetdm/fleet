# frozen_string_literal: true

require 'puppet/util/fleet_client'

Puppet::Functions.create_function(:"fleetdm::preassign_profile") do
  dispatch :preassign_profile do
    param 'String', :uuid
    param 'String', :template
    optional_param 'String', :group
  end

  def preassign_profile(uuid, template, group = 'default')
    host = call_function('lookup', 'fleetdm::host')
    token = call_function('lookup', 'fleetdm::token')
    client = Puppet::Util::FleetClient.new(host, token)
    client.preassign_profile(uuid, template, group)
  end
end
