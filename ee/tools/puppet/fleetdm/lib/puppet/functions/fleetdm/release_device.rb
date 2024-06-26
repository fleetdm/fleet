# frozen_string_literal: true

require_relative '../../util/fleet_client'

# fleetdm::release_device sends the [`DeviceConfigured`][1] MDM command to the
# device with the provided UUID. This is useful to release DEP enrolled devices
# during setup.
#
# [1]: https://developer.apple.com/documentation/devicemanagement/release_device_from_await_configuration
Puppet::Functions.create_function(:"fleetdm::release_device") do
  dispatch :release_device do
    param 'String', :uuid
  end

  def release_device(uuid)
    command_xml = <<~COMMAND_TEMPLATE
      <?xml version="1.0" encoding="UTF-8"?>
      <!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
      <plist version="1.0">
        <dict>
          <key>Command</key>
          <dict>
            <key>RequestType</key>
            <string>DeviceConfigured</string>
          </dict>
          <key>CommandUUID</key>
          <string>#{SecureRandom.uuid}</string>
        </dict>
      </plist>
    COMMAND_TEMPLATE

    env = closure_scope['environment']
    client = Puppet::Util::FleetClient.instance
    response = client.send_mdm_command(uuid, command_xml, env)

    if response['error'].empty?
      Puppet.info('successfully released device')
    else
      Puppet.err("error releasing device: #{response['error']}")
    end

    response
  end
end
