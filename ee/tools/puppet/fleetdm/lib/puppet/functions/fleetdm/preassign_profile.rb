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
    return check_for_error(client_resp, "Error pre-assigning profile #{profile_identifier} (ensure #{ensure_profile})", template, preassign_profile_response) if client_resp&.[]('error')&.present?

    # get host by idenfifier to get the host id
    client_resp = client.get_host_by_identifier(host_uuid, env)
    return check_for_error(client_resp, "Error getting host by identifier #{host_uuid}", template, preassign_profile_response) if client_resp&.[]('error')&.present?

    unless client_resp['body'] && client_resp['body']['host'] && client_resp['body']['host']['id']
      return handle_error("No host found for #{host_uuid}", client_resp['error'], template, preassign_profile_response)
    end

    # get host profiles currently assigned to the host
    client_resp = client.get_host_profiles(client_resp['body']['host']['id'], env)
    return check_for_error(client_resp, "Error getting host profiles for #{host_uuid}", template, preassign_profile_response) if client_resp&.[]('error')&.present?

    # if this is the first run on the device, profiles will be empty so we can skip the checksum
    # comparison and mark the resource as changed depending on the ensure_profile value
    unless client_resp['body'] && client_resp['body']['profiles'] && !client_resp['body']['profiles']&.empty?
      Puppet.info("No assigned profiles found, this may be the first run for #{host_uuid}")
      preassign_profile_response['resource_changed'] = ensure_profile == 'present'
      return preassign_profile_response
    end

    # compare checksums to see if the profile is already assigned
    base64_checksum = Digest::MD5.base64digest(template)
    has_profile = client_resp['body']['profiles'].any? { |p| p['checksum'] == base64_checksum }
    if (has_profile && ensure_profile == 'absent') || (!has_profile && ensure_profile == 'present')
      preassign_profile_response['resource_changed'] = true
    else
      Puppet.info("Profile #{profile_identifier} already #{ensure_profile} for #{host_uuid}")
    end

    preassign_profile_response
  end

  private

  def handle_error(message, error, template, preassign_profile_response)
    Puppet.err("#{message}: #{error} \n\n #{template}")
    preassign_profile_response['error'] = error
    preassign_profile_response
  end

  def check_for_error(response, error_message, template, preassign_profile_response)
    handle_error(error_message, response&.[]('error'), template, preassign_profile_response) if response&.[]('error')&.present?
  end
end
