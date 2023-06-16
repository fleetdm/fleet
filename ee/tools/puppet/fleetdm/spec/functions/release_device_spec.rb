# frozen_string_literal: true

require 'spec_helper'
require 'puppet/util/fleet_client'

describe 'fleetdm::release_device' do
  let(:fleet_client_mock) { instance_double('Puppet::Util::FleetClient') }
  let(:device_uuid) { 'device-uuid' }

  before(:each) do
    fleet_client_class = class_spy('Puppet::Util::FleetClient')
    stub_const('Puppet::Util::FleetClient', fleet_client_class)
    allow(fleet_client_class).to receive(:new).with('https://example.com', 'test_token') { fleet_client_mock }
  end

  it { is_expected.to run.with_params(nil).and_raise_error(StandardError) }

  it 'performs an API call to Fleet' do
    expect(fleet_client_mock).to receive(:send_mdm_command).with(device_uuid, %r{DeviceConfigured})
    is_expected.to run.with_params(device_uuid)
  end
end
