# frozen_string_literal: true

require 'puppet/util/fleet_client'

# fleetdm::command_xml sends a custom MDM command to the device
# with the provided UUID.
#
# For more information on MDM commands and queries, refer to the Apple Developer documentation:
# https://developer.apple.com/documentation/devicemanagement/commands_and_queries
Puppet::Functions.create_function(:"fleetdm::command_xml") do
  dispatch :command_xml do
    param 'String', :uuid
    param 'String', :xml_data
  end

  def command_xml(uuid, xml_data)
    host = call_function('lookup', 'fleetdm::host')
    token = call_function('lookup', 'fleetdm::token')
    client = Puppet::Util::FleetClient.new(host, token)
    response = client.send_mdm_command(uuid, xml_data)

    if response['error'].empty?
      Puppet.info('Successfully sent custom MDM command')
    else
      Puppet.err("Error sending custom MDM command: #{response['error']}")
    end

    response
  end
end
