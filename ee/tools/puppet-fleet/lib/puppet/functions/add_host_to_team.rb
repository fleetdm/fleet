# frozen_string_literal: true

require_relative '../util/fleet_client'

Puppet::Functions.create_function(:add_host_to_team) do
  def add_host_to_team(host_uuid, team_name, fleet_host, fleet_token)
    client = Puppet::Util::FleetClient.new(fleet_host, fleet_token)
    team_resp = client.team_id_from_name(team_name)
    return team_resp if team_resp['error']

    client.transfer_host(team_resp['output']['teams'][0]['id'], host_uuid)
  end
end
