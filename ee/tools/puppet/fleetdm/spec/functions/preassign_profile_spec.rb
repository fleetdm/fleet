# frozen_string_literal: true

require 'spec_helper'

describe 'fleetdm::preassign_profile' do
  let(:fleet_client_mock) { instance_double('Puppet::Util::FleetClient') }
  let(:device_uuid) { 'device-uuid' }
  let(:template) { 'template' }
  let(:group) { 'group' }
  let(:catalog_uuid) { '827a74c8-cf98-44da-9ff7-18c5e4bee41e' }
  let(:profile_identifier) { 'test.example.com' }

  before(:each) do
    fleet_client_class = class_spy('Puppet::Util::FleetClient')
    stub_const('Puppet::Util::FleetClient', fleet_client_class)
    allow(fleet_client_class).to receive(:new).with('https://example.com', 'test_token') { fleet_client_mock }
    allow(SecureRandom).to receive(:uuid).and_return(catalog_uuid)
  end

  it { is_expected.to run.with_params(nil).and_raise_error(StandardError) }

  it 'performs an API call to Fleet with the right parameters' do
    expect(fleet_client_mock).to receive(:preassign_profile).with(catalog_uuid, device_uuid, template, group).and_return({ 'error' => '' })
    is_expected.to run.with_params(profile_identifier, device_uuid, template, group)
  end

  it 'has a default value if group is not provided' do
    expect(fleet_client_mock).to receive(:preassign_profile).with(catalog_uuid, device_uuid, template, 'default').and_return({ 'error' => '' })
    is_expected.to run.with_params(profile_identifier, device_uuid, template)
  end
end
