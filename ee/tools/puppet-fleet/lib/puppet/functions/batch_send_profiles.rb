# frozen_string_literal: true

require 'base64'
require_relative '../util/fleet_client'

Puppet::Functions.create_function(:batch_send_profiles) do
  def batch_send_profiles(team_name, profiles, fleet_host, fleet_token)
    enc = profiles.map { |p| Base64.encode64(p) }
    client = Puppet::Util::FleetClient.new(fleet_host, fleet_token)
    client.batch_send_profiles(team_name, enc)
  end
end
