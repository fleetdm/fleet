# frozen_string_literal: true

require 'spec_helper'

describe 'fleetdm::preassign_profile' do
  let(:fleet_client_mock) { instance_double('Puppet::Util::FleetClient') }
  let(:device_uuid) { 'device-uuid' }
  let(:template) { 'template' }
  let(:group) { 'group' }
  let(:ensure_profile) { 'absent' }
  let(:node_name) { Puppet[:node_name_value] }
  let(:catalog_uuid) { '827a74c8-cf98-44da-9ff7-18c5e4bee41e' }
  let(:run_identifier) { "#{catalog_uuid}-#{node_name}" }
  let(:profile_identifier) { 'test.example.com' }
  let(:host_response) { { 'host' => { 'id' => 1 } } }
  let(:rspec_puppet_env) { 'rp_env' }

  before(:each) do
    fleet_client_class = class_spy('Puppet::Util::FleetClient')
    stub_const('Puppet::Util::FleetClient', fleet_client_class)
    allow(fleet_client_class).to receive(:instance) { fleet_client_mock }
    allow(SecureRandom).to receive(:uuid).and_return(catalog_uuid)
  end

  it { is_expected.to run.with_params(nil).and_raise_error(StandardError) }

  it 'performs an API call to Fleet with the right parameters' do
    expect(fleet_client_mock)
      .to receive(:get_host_by_identifier)
      .with(device_uuid, rspec_puppet_env)
      .and_return({ 'error' => '', 'body' => host_response })
    expect(fleet_client_mock)
      .to receive(:get_host_profiles)
      .with(host_response['host']['id'], rspec_puppet_env)
      .and_return({ 'error' => '', 'body' => { 'profiles' => [] } })
    expect(fleet_client_mock)
      .to receive(:preassign_profile)
      .with(run_identifier, device_uuid, template, group, ensure_profile, rspec_puppet_env)
      .and_return({ 'error' => '' })
    is_expected.to run.with_params(profile_identifier, device_uuid, template, group, ensure_profile)
  end

  it 'has default values for `group` and `ensure`' do
    expect(fleet_client_mock)
      .to receive(:get_host_by_identifier)
      .with(device_uuid, rspec_puppet_env)
      .and_return({ 'error' => '', 'body' => host_response })
    expect(fleet_client_mock)
      .to receive(:get_host_profiles)
      .with(host_response['host']['id'], rspec_puppet_env)
      .and_return({ 'error' => '', 'body' => { 'profiles' => [] } })
    expect(fleet_client_mock)
      .to receive(:preassign_profile)
      .with(run_identifier, device_uuid, template, 'default', 'present', rspec_puppet_env)
      .and_return({ 'error' => '' })
    is_expected.to run.with_params(profile_identifier, device_uuid, template)
  end

  #   TODO: add coverage for early exits, error handling, and resource_changed
  it 'gracefully handles preassign_profile failures' do
    expect(fleet_client_mock)
      .to receive(:preassign_profile)
      .with(run_identifier, device_uuid, template, 'default', 'present', rspec_puppet_env)
      .and_return({ 'error' => 'preassign error' })
    is_expected.to run.with_params(profile_identifier, device_uuid, template).and_return({ 'error' => 'preassign error', 'resource_changed' => false })
  end

  it 'gracefully handles get_host_by_identifier failures' do
    expect(fleet_client_mock)
      .to receive(:preassign_profile)
      .with(run_identifier, device_uuid, template, 'default', 'present', rspec_puppet_env)
      .and_return({ 'error' => '' })
    expect(fleet_client_mock)
      .to receive(:get_host_by_identifier)
      .with(device_uuid, rspec_puppet_env)
      .and_return({ 'error' => 'get host by identifier error' })
    is_expected.to run.with_params(profile_identifier, device_uuid, template).and_return({ 'error' => 'get host by identifier error', 'resource_changed' => false })
  end

  it 'gracefully handles get_host_profiles failures' do
    expect(fleet_client_mock)
      .to receive(:preassign_profile)
      .with(run_identifier, device_uuid, template, 'default', 'present', rspec_puppet_env)
      .and_return({ 'error' => '' })
    expect(fleet_client_mock)
      .to receive(:get_host_by_identifier)
      .with(device_uuid, rspec_puppet_env)
      .and_return({ 'error' => '', 'body' => host_response })
    expect(fleet_client_mock)
      .to receive(:get_host_profiles)
      .with(host_response['host']['id'], rspec_puppet_env)
      .and_return({ 'error' => 'get host profiles error' })
    is_expected.to run.with_params(profile_identifier, device_uuid, template).and_return({ 'error' => 'get host profiles error', 'resource_changed' => false })
  end

  it 'handles the case where no host is found' do
    expect(fleet_client_mock)
      .to receive(:preassign_profile)
      .with(run_identifier, device_uuid, template, 'default', 'present', rspec_puppet_env)
      .and_return({ 'error' => '' })
    expect(fleet_client_mock)
      .to receive(:get_host_by_identifier)
      .with(device_uuid, rspec_puppet_env)
      .and_return({ 'error' => '', 'body' => {} }) # Simulating no host found
    is_expected.to run.with_params(profile_identifier, device_uuid, template).and_return({ 'error' => 'No host found', 'resource_changed' => false })
  end

  it 'does not change resource when profile is already present and ensure is present' do
    host_profiles = { 'profiles' => [{ 'checksum' => Digest::MD5.base64digest(template) }] }
    expect_successful_calls_with_host_profiles(host_profiles, 'present')
    is_expected.to run.with_params(profile_identifier, device_uuid, template).and_return({ 'error' => '', 'resource_changed' => false })
  end

  it 'does not change resource when profile is absent and ensure is absent' do
    host_profiles = { 'profiles' => [] } # no profiles assigned
    expect_successful_calls_with_host_profiles(host_profiles, 'absent')
    is_expected.to run.with_params(profile_identifier, device_uuid, template, 'default', 'absent').and_return({ 'error' => '', 'resource_changed' => false })
  end

  it 'changes resource when profile is present and ensure is absent' do
    host_profiles = { 'profiles' => [{ 'checksum' => Digest::MD5.base64digest(template) }] }
    expect_successful_calls_with_host_profiles(host_profiles, 'absent')
    is_expected.to run.with_params(profile_identifier, device_uuid, template, 'default', 'absent').and_return({ 'error' => '', 'resource_changed' => true })
  end

  it 'changes resource when profile is absent and ensure is present' do
    host_profiles = { 'profiles' => [] } # No profiles assigned
    expect_successful_calls_with_host_profiles(host_profiles, 'present')
    is_expected.to run.with_params(profile_identifier, device_uuid, template, 'default', 'present').and_return({ 'error' => '', 'resource_changed' => true })
  end

  def expect_successful_calls_with_host_profiles(host_profiles, ensure_profile)
    expect(fleet_client_mock)
      .to receive(:preassign_profile)
      .with(run_identifier, device_uuid, template, 'default', ensure_profile, rspec_puppet_env)
      .and_return({ 'error' => '' })
    expect(fleet_client_mock)
      .to receive(:get_host_by_identifier)
      .with(device_uuid, rspec_puppet_env)
      .and_return({ 'error' => '', 'body' => host_response })
    expect(fleet_client_mock)
      .to receive(:get_host_profiles)
      .with(host_response['host']['id'], rspec_puppet_env)
      .and_return({ 'error' => '', 'body' => host_profiles })
  end
end
