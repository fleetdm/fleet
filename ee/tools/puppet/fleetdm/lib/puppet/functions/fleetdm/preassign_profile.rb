# frozen_string_literal: true

require 'puppet/util/fleet_client'

Puppet::Functions.create_function(:"fleetdm::preassign_profile") do
  dispatch :preassign_profile do
    param 'String', :profile_identifier
    param 'String', :host_uuid
    param 'String', :template
    optional_param 'String', :group
  end

  def preassign_profile(profile_identifier, host_uuid, template, group = 'default')
    host = call_function('lookup', 'fleetdm::host')
    token = call_function('lookup', 'fleetdm::token')
    client = Puppet::Util::FleetClient.new(host, token)
    response = client.preassign_profile(host_uuid, template, group)

    if response['error'].empty?
      Puppet.info("successfully pre-assigned profile #{profile_identifier}")
    else
      Puppet.err("error pre-assigning profile #{profile_identifier}: #{response['error']}")
    end

    response
  end
end
