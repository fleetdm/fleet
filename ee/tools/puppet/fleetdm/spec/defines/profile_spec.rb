# frozen_string_literal: true

require 'spec_helper'

describe 'fleetdm::profile' do
  let(:fleet_client_mock) { instance_double('Puppet::Util::FleetClient') }
  let(:title) { 'namevar' }
  let(:template) { 'test-template' }
  let(:group) { 'group' }
  let(:node) { 'testhost.example.com' }
  let(:params) do
    { 'template' => template, 'group' => group }
  end

  before(:each) do
    fleet_client_class = class_spy('Puppet::Util::FleetClient')
    stub_const('Puppet::Util::FleetClient', fleet_client_class)
    allow(fleet_client_class).to receive(:new).with('https://example.com', 'test_token') { fleet_client_mock }
  end

  on_supported_os.each do |os, os_facts|
    context "on #{os}" do
      let(:facts) { os_facts }

      it 'compiles' do
        uuid = os_facts[:system_profiler]['hardware_uuid']
        expect(fleet_client_mock).to receive(:preassign_profile).with(uuid, template, group)
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

      context 'without group' do
        let(:params) do
          { 'template' => template }
        end

        it 'compiles' do
          uuid = os_facts[:system_profiler]['hardware_uuid']
          expect(fleet_client_mock).to receive(:preassign_profile).with(uuid, template, 'default')
          is_expected.to compile
        end
      end
    end
  end
end
