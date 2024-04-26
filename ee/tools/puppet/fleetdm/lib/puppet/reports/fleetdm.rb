# frozen_string_literal: true

require 'puppet'
require_relative '../util/fleet_client'

Puppet::Reports.register_report(:fleetdm) do
  desc 'Used to signal the Fleet server that a Puppet run is done to match a device to a team for profile delivery'

  def process
    return if noop

    node_name = Puppet[:node_name_value]
    if resource_statuses.any? { |r, _| r.downcase.include?('error pre-setting fleetdm::profile') }
      Puppet.err("Some resources failed to be assigned, not matching profiles for #{node_name}")
      return
    end

    client = Puppet::Util::FleetClient.instance
    run_identifier = "#{catalog_uuid}-#{node_name}"
    response = client.match_profiles(run_identifier, environment)
    if response['error'].empty?
      Puppet.info("Successfully matched #{node_name} with a team containing configuration profiles")
      return
    end

    Puppet.err("Error matching node #{node_name} with a team containing configuration profiles: #{response['error']}")
  end
end
