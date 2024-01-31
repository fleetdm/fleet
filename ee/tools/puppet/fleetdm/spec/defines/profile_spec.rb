# frozen_string_literal: true

require 'spec_helper'

describe 'fleetdm::profile' do
  let(:fleet_client_mock) { instance_double('Puppet::Util::FleetClient') }
  let(:title) { 'namevar' }
  let(:template) { 'test-template' }
  let(:group) { 'group' }
  let(:node_name) { Puppet[:node_name_value] }
  let(:catalog_uuid) { '827a74c8-cf98-44da-9ff7-18c5e4bee41e' }
  let(:run_identifier) { "#{catalog_uuid}-#{node_name}" }
  let(:host_response) { { 'host' => { 'id' => 1 } } }
  let(:params) do
    { 'template' => template, 'group' => group }
  end

  before(:each) do
    fleet_client_class = class_spy('Puppet::Util::FleetClient')
    stub_const('Puppet::Util::FleetClient', fleet_client_class)
    allow(fleet_client_class).to receive(:instance) { fleet_client_mock }
    allow(SecureRandom).to receive(:uuid).and_return(catalog_uuid)
  end

  on_supported_os.each do |os, os_facts|
    context "on #{os}" do
      let(:facts) { os_facts.merge({}) }

      it 'compiles' do
        uuid = os_facts[:system_profiler]['hardware_uuid']
        expect(fleet_client_mock)
          .to receive(:get_host_by_identifier)
          .with(uuid, 'production')
          .and_return({ 'error' => '', 'body' => host_response })
        expect(fleet_client_mock)
          .to receive(:get_host_profiles)
          .with(host_response['host']['id'], 'production')
          .and_return({ 'error' => '', 'body' => { 'profiles' => [] } })
        expect(fleet_client_mock)
          .to receive(:preassign_profile)
          .with(run_identifier, uuid, template, group, 'present', 'production')
          .and_return({ 'error' => '' })
        is_expected.to compile
      end

      context 'noop' do
        let(:facts) { { 'clientnoop' => true } }

        it 'does not send a request in noop mode' do
          is_expected.to compile
        end
      end

      context 'invalid template' do
        let(:params) do
          { 'template' => '', 'group' => group }
        end

        it { is_expected.to compile.and_raise_error(%r{invalid template}) }
      end

      context 'invalid group' do
        let(:params) do
          { 'template' => template, 'group' => '' }
        end

        it { is_expected.to compile.and_raise_error(%r{invalid group}) }
      end

      context 'invalid ensure' do
        let(:params) do
          { 'template' => template, 'ensure' => 'nothing' }
        end

        it { is_expected.to compile.and_raise_error(%r{'ensure' expects a match for Enum\['absent', 'present'\]}) }
      end

      context 'without group' do
        let(:params) do
          { 'template' => template }
        end

        it 'compiles' do
          uuid = os_facts[:system_profiler]['hardware_uuid']
          expect(fleet_client_mock)
            .to receive(:get_host_by_identifier)
            .with(uuid, 'production')
            .and_return({ 'error' => '', 'body' => host_response })
          expect(fleet_client_mock)
            .to receive(:get_host_profiles)
            .with(host_response['host']['id'], 'production')
            .and_return({ 'error' => '', 'body' => { 'profiles' => [] } })
          expect(fleet_client_mock)
            .to receive(:preassign_profile)
            .with(run_identifier, uuid, template, 'default', 'present', 'production')
            .and_return({ 'error' => '' })
          is_expected.to compile
        end
      end

      context 'ensure => absent' do
        let(:params) do
          { 'template' => template, 'ensure' => 'absent' }
        end

        it 'compiles' do
          uuid = os_facts[:system_profiler]['hardware_uuid']
          expect(fleet_client_mock)
            .to receive(:get_host_by_identifier)
            .with(uuid, 'production')
            .and_return({ 'error' => '', 'body' => host_response })
          expect(fleet_client_mock)
            .to receive(:get_host_profiles)
            .with(host_response['host']['id'], 'production')
            .and_return({ 'error' => '', 'body' => { 'profiles' => [] } })
          expect(fleet_client_mock)
            .to receive(:preassign_profile)
            .with(run_identifier, uuid, template, 'default', 'absent', 'production')
            .and_return({ 'error' => '' })
          is_expected.to compile
        end
      end
    end
  end
end
