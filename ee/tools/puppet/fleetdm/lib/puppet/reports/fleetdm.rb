# frozen_string_literal: true

require 'puppet'
require 'puppet/util/fleet_client'
require 'puppet/util/env'

Puppet::Reports.register_report(:fleetdm) do
  desc 'Used to signal the Fleet server that a Puppet run is done to match a device to a team for profile delivery'

  def process
    return if noop
    client = Puppet::Util::FleetClient.instance
    node_name = Puppet[:node_name_value]
    run_identifier = "#{catalog_uuid}-#{node_name}"
    response = client.match_profiles(run_identifier)

    if response['error'].empty?
      Puppet.info("successfully matched #{node_name} with a team containing configuration profiles")
    else
      Puppet.err("error matching node #{node_name} with a team containing configuration profiles: #{response['error']}")

      error_webhook_url = Puppet::Util.read_hiera('fleetdm::error_webhook_url')
      return unless error_webhook_url

      uri = URI(error_webhook_url)
      res = Net::HTTP.post_form(uri, 'title' => 'foo', 'body' => 'bar', 'userID' => 1)

      unless res.is_a?(Net::HTTPSuccess)
        Puppet.err("error sending webhook in reporter: #{res}")
      end
    end
  end
end
