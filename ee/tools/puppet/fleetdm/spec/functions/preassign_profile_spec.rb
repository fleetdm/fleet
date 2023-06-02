# frozen_string_literal: true

require 'spec_helper'

describe 'fleetdm::preassign_profile' do
  let(:fleet_client_mock) { instance_double('Puppet::Util::FleetClient') }
  let(:device_uuid) { 'device-uuid' }
  let(:template) { 'template' }
  let(:group) { 'group' }

  before(:each) do
    fleet_client_class = class_spy('Puppet::Util::FleetClient')
    stub_const('Puppet::Util::FleetClient', fleet_client_class)
    allow(fleet_client_class).to receive(:new).with('https://example.com', 'test_token') { fleet_client_mock }
  end

  it { is_expected.to run.with_params(nil).and_raise_error(StandardError) }

  it 'performs an API call to Fleet with the right parameters' do
    expect(fleet_client_mock).to receive(:preassign_profile).with(device_uuid, template, group)
    is_expected.to run.with_params(device_uuid, template, group)
  end

  it 'has a default value if group is not provided' do
    expect(fleet_client_mock).to receive(:preassign_profile).with(device_uuid, template, 'default')
    is_expected.to run.with_params(device_uuid, template)
  end
end
