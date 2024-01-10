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
    # initialize our response
    preassign_profile_response = { 'error' => '', 'resource_changed' => false }

    client = Puppet::Util::FleetClient.instance
    env = closure_scope['server_facts']['environment']
    run_identifier = "#{closure_scope.catalog.catalog_uuid}-#{Puppet[:node_name_value]}"

    # initiate the pre-assignment process with fleet server
    client_resp = client.preassign_profile(run_identifier, host_uuid, template, group, ensure_profile, closure_scope['environment'])
    unless client_resp && client_resp['error'].empty?
      Puppet.err("error pre-assigning profile #{profile_identifier} (ensure #{ensure_profile}): #{client_resp&.[]('error')} \n\n #{template}")
      preassign_profile_response['error'] = client_resp['error']
      return preassign_profile_response
    end

    # get host by idenfifier to get the host id
    client_resp = client.get_host_by_identifier(host_uuid, env)
    unless client_resp && client_resp['error'].empty?
      Puppet.err("error getting host by identifier #{host_uuid}: #{client_resp&.[]('error')} \n\n #{template}")
      preassign_profile_response['error'] = client_resp['error']
      return preassign_profile_response
    end
    unless client_resp['body'] && client_resp['body']['host'] && client_resp['body']['host']['id']
      Puppet.err("no host found for #{host_uuid} \n\n #{template}")
      preassign_profile_response['error'] = client_resp['error']
      return preassign_profile_response
    end

    # get host profiles currently assigned to the host
    client_resp = client.get_host_profiles(client_resp['body']['host']['id'], env)
    unless client_resp && client_resp['error'].empty?
      Puppet.err("error getting host profiles for #{host_uuid}: #{client_resp&.[]('error')} \n\n #{template}")
      preassign_profile_response['error'] = client_resp['error']
      return preassign_profile_response
    end
    unless client_resp['body'] && client_resp['body']['profiles'] && !client_resp['body']['profiles'].empty?
      Puppet.err("no profiles returned for #{host_uuid} \n\n #{template}")
      preassign_profile_response['error'] = client_resp['error']
      return preassign_profile_response
    end

    # compare checksums to see if the profile is already assigned
    base64_checksum = Digest::MD5.base64digest(template)
    has_profile = client_resp['body']['profiles'].any? { |p| p['checksum'] == base64_checksum }
    if (has_profile && ensure_profile == 'absent') || (!has_profile && ensure_profile == 'present')
      preassign_profile_response['resource_changed'] = true
    else
      Puppet.info("profile_identifier #{ profile_identifier } already #{ ensure_profile } for #{ host_uuid }")
    end

    preassign_profile_response
  end
end
