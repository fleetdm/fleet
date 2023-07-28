# frozen_string_literal: true

require 'puppet'
require_relative '../util/fleet_client'

Puppet::Reports.register_report(:fleetdm) do
  desc 'Used to signal the Fleet server that a Puppet run is done to match a device to a team for profile delivery'

  def process
    return if noop
    client = Puppet::Util::FleetClient.instance
    node_name = Puppet[:node_name_value]
    run_identifier = "#{catalog_uuid}-#{node_name}"
    response = client.match_profiles(run_identifier, environment)

    if response['error'].empty?
      Puppet.info("successfully matched #{node_name} with a team containing configuration profiles")
    else
      Puppet.err("error matching node #{node_name} with a team containing configuration profiles: #{response['error']}")
    end
  end
end
