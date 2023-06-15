# frozen_string_literal: true

require 'puppet'
require 'puppet/util/fleet_client'

Puppet::Reports.register_report(:fleetdm) do
  desc 'Used to signal the Fleet server that a Puppet run is done to match a device to a team for profile delivery'

  def process
    return if noop
    node = Puppet::Node.new(Puppet[:node_name_value])
    compiler = Puppet::Parser::Compiler.new(node)
    scope = Puppet::Parser::Scope.new(compiler)
    lookup_invocation = Puppet::Pops::Lookup::Invocation.new(scope, {}, {}, nil)
    host = Puppet::Pops::Lookup.lookup('fleetdm::host', nil, '', false, nil, lookup_invocation)
    token = Puppet::Pops::Lookup.lookup('fleetdm::token', nil, '', false, nil, lookup_invocation)

    client = Puppet::Util::FleetClient.new(host, token)
    response = client.match_profiles

    if response['error'].empty?
      Puppet.info("successfully matched #{node} with a team containing configuration profiles")
    else
      Puppet.err("error matching node #{node} with a team containing configuration profiles: #{response['error']}")
    end
  end
end
