# frozen_string_literal: true

require 'spec_helper'
require_relative '../../lib/puppet/util/fleet_client.rb'

describe 'fleetdm::profile' do
  let(:fleet_client_mock) { instance_double('Puppet::Util::FleetClient') }
  let(:title) { 'namevar' }
  let(:template) { 'test-template' }
  let(:group) { 'group' }
  let(:node_name) { Puppet[:node_name_value] }
  let(:catalog_uuid) { '827a74c8-cf98-44da-9ff7-18c5e4bee41e' }
  let(:run_identifier) { "#{catalog_uuid}-#{node_name}" }
  let(:host_response) { { 'host' => { 'id' => 1 } } }
  let(:rspec_puppet_env) { 'rp_env' }

  before(:each) do
    allow(Puppet::Util::FleetClient).to receive(:instance).and_return(fleet_client_mock)
    allow(SecureRandom).to receive(:uuid).and_return(catalog_uuid)
  end

  on_supported_os.each do |os, os_facts|
    context "on #{os}" do
      let(:facts) { os_facts.merge({}) }

      context 'noop' do
        let(:params) do
          { 'template' => template, 'group' => group }
        end
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

      # FIXME(roberto): for some reason I don't understand, the first call to
      # any method that uses Puppet::Util::FleetClient always uses the real
      # class instead of the double, but all subsequent calls do.
      #
      # My theory is that this class is pre-loaded by some mechanism that I
      # haven't figured out how to tweak. This hack is a "dummy" test that does
      # nothing and the only purpose is to cleanup that pre-loaded version.
      context 'clean preloaded mock' do
        let(:params) do
          { 'template' => template, 'group' => group }
        end

        it 'compiles' do
          expect(fleet_client_mock)
            .not_to receive(:get_host_by_identifier)
          expect(fleet_client_mock)
            .not_to receive(:get_host_profiles)
          expect(fleet_client_mock)
            .not_to receive(:preassign_profile)

          is_expected.to compile
        end
      end

      context 'with different template' do
        let(:params) do
          { 'template' => 'foo', 'group' => group }
        end

        it 'compiles 2' do
          uuid = os_facts[:system_profiler]['hardware_uuid']
          expect(fleet_client_mock)
            .to receive(:get_host_by_identifier)
            .with(uuid, rspec_puppet_env)
            .and_return({ 'error' => '', 'body' => host_response })
          expect(fleet_client_mock)
            .to receive(:get_host_profiles)
            .with(host_response['host']['id'], rspec_puppet_env)
            .and_return({ 'error' => '', 'body' => { 'profiles' => [] } })
          expect(fleet_client_mock)
            .to receive(:preassign_profile)
            .with(run_identifier, uuid, 'foo', group, 'present', rspec_puppet_env)
            .and_return({ 'error' => '' })

          is_expected.to compile
        end
      end

      context 'without group' do
        let(:params) do
          { 'template' => template }
        end

        it 'compiles' do
          uuid = os_facts[:system_profiler]['hardware_uuid']
          expect(fleet_client_mock)
            .to receive(:preassign_profile)
            .with(run_identifier, uuid, template, 'default', 'present', rspec_puppet_env)
            .and_return({ 'error' => '' })
          expect(fleet_client_mock)
            .to receive(:get_host_by_identifier)
            .with(uuid, rspec_puppet_env)
            .and_return({ 'error' => '', 'body' => host_response })
          expect(fleet_client_mock)
            .to receive(:get_host_profiles)
            .with(host_response['host']['id'], rspec_puppet_env)
            .and_return({ 'error' => '', 'body' => { 'profiles' => [] } })
          is_expected.to compile
        end
      end

      context 'ensure => absent' do
        let(:facts) { os_facts.merge({}) }
        let(:params) do
          { 'template' => template, 'ensure' => 'absent' }
        end

        it 'compiles' do
          uuid = os_facts[:system_profiler]['hardware_uuid']
          expect(fleet_client_mock)
            .to receive(:get_host_by_identifier)
            .with(uuid, rspec_puppet_env)
            .and_return({ 'error' => '', 'body' => host_response })
          expect(fleet_client_mock)
            .to receive(:get_host_profiles)
            .with(host_response['host']['id'], rspec_puppet_env)
            .and_return({ 'error' => '', 'body' => { 'profiles' => [] } })
          expect(fleet_client_mock)
            .to receive(:preassign_profile)
            .with(run_identifier, uuid, template, 'default', 'absent', rspec_puppet_env)
            .and_return({ 'error' => '' })
          is_expected.to compile
        end
      end
    end
  end
end
