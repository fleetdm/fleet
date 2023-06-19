# frozen_string_literal: true

require 'puppet'
require 'puppet/util/fleet_client'

Puppet::Reports.register_report(:fleetdm) do
  desc 'Used to signal the Fleet server that a Puppet run is done to match a device to a team for profile delivery'

  def process
    return if noop
    node_name = Puppet[:node_name_value]
    node = Puppet::Node.new(node_name)
    compiler = Puppet::Parser::Compiler.new(node)
    scope = Puppet::Parser::Scope.new(compiler)
    lookup_invocation = Puppet::Pops::Lookup::Invocation.new(scope, {}, {}, nil)
    host = Puppet::Pops::Lookup.lookup('fleetdm::host', nil, '', false, nil, lookup_invocation)
    token = Puppet::Pops::Lookup.lookup('fleetdm::token', nil, '', false, nil, lookup_invocation)

    client = Puppet::Util::FleetClient.new(host, token)
    run_identifier = catalog_uuid || node_name
    response = client.match_profiles(run_identifier)

    if response['error'].empty?
      Puppet.info("successfully matched #{node_name} with a team containing configuration profiles")
    else
      Puppet.err("error matching node #{node_name} with a team containing configuration profiles: #{response['error']}")
    end
  end
end
