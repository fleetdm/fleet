# frozen_string_literal: true

require 'spec_helper'
require_relative '../../lib/puppet/util/fleet_client.rb'

describe 'fleetdm::release_device' do
  let(:fleet_client_mock) { instance_double('Puppet::Util::FleetClient') }
  let(:device_uuid) { 'device-uuid' }
  let(:rspec_puppet_env) { 'rp_env' }

  before(:each) do
    allow(Puppet::Util::FleetClient).to receive(:instance).and_return(fleet_client_mock)
  end

  on_supported_os.each do |os, os_facts|
    context "on #{os}" do
      let(:facts) { os_facts.merge({}) }

      it { is_expected.to run.with_params(nil).and_raise_error(StandardError) }

      it 'performs an API call to Fleet' do
        expect(fleet_client_mock).to receive(:send_mdm_command) { |device_uuid_param, command_param, environment_param|
          expect(device_uuid_param).to eq(device_uuid)
          expect(command_param).to include('DeviceConfigured')
          expect(environment_param).to eq(rspec_puppet_env)
        }.and_return({ 'error' => '' })

        is_expected.to run.with_params(device_uuid)
      end
    end
  end
end
