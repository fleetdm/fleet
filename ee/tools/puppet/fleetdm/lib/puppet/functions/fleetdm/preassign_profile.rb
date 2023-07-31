# frozen_string_literal: true

require_relative '../../util/fleet_client'

Puppet::Functions.create_function(:"fleetdm::preassign_profile") do
  dispatch :preassign_profile do
    param 'String', :profile_identifier
    param 'String', :host_uuid
    param 'String', :template
    optional_param 'String', :group
    optional_param 'Enum[absent, present]', :ensure
  end

  def preassign_profile(profile_identifier, host_uuid, template, group = 'default', ensure_profile = 'present')
    client = Puppet::Util::FleetClient.instance
    env = closure_scope['server_facts']['environment']
    run_identifier = "#{closure_scope.catalog.catalog_uuid}-#{Puppet[:node_name_value]}"
    response = client.preassign_profile(run_identifier, host_uuid, template, group, ensure_profile, closure_scope['environment'])

    if response['error'].empty?
      base64_checksum = Digest::MD5.base64digest(template)
      host = client.get_host_by_identifier(host_uuid, env)
      host_profiles = client.get_host_profiles(host['body']['host']['id'], env)

      if host_profiles['error'].empty?
        Puppet.info("successfully pre-set profile #{profile_identifier} as #{ensure_profile}")

        has_profile = host_profiles['body']['profiles'].any? { |p| p['checksum'] == base64_checksum }
        if (has_profile && ensure_profile == 'absent') || (!has_profile && ensure_profile == 'present')
          response['resource_changed'] = true
        end
      end
    else
      Puppet.err("error pre-setting profile #{profile_identifier} (ensure #{ensure_profile}): #{response['error']} \n\n #{template}")
    end

    response
  end
end
