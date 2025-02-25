# frozen_string_literal: true

require 'spec_helper'
require 'puppet/reports'
require_relative '../../../lib/puppet/reports/fleetdm.rb'

describe 'Puppet::Reports::Fleetdm' do
  let(:fleet_client_mock) { instance_double('Puppet::Util::FleetClient') }
  let(:catalog_uuid) { '827a74c8-cf98-44da-9ff7-18c5e4bee41e' }
  let(:node_name) { Puppet[:node_name_value] }
  let(:report) do
    report = Puppet::Transaction::Report.new('apply')
    report.extend(Puppet::Reports.report(:fleetdm))
    report
  end

  before(:each) do
    Puppet[:reports] = 'fleetdm'
    Puppet::Util::Log.level = :warning
    Puppet::Util::Log.newdestination(:console)

    fleet_client_class = class_spy('Puppet::Util::FleetClient')
    stub_const('Puppet::Util::FleetClient', fleet_client_class)
    allow(fleet_client_class).to receive(:instance) { fleet_client_mock }
    allow(SecureRandom).to receive(:uuid).and_return(catalog_uuid)
  end

  it 'does not process in noop mode' do
    allow(report).to receive(:noop).and_return(true)
    expect(fleet_client_mock).not_to receive(:match_profiles)
    report.process
  end

  it 'logs an error if resources failed to be assigned' do
    allow(report).to receive(:resource_statuses).and_return({ 'error pre-setting fleetdm::profile com.apple.SoftwareUpdate as present: forbidden : base forbidden' => 'anything' })
    expect(Puppet).to receive(:err).with(%r{Some resources failed to be assigned})
    expect(fleet_client_mock).not_to receive(:match_profiles)
    report.process
  end

  it 'successfully matches profiles when there are no errors' do
    allow(report).to receive(:noop).and_return(false)
    allow(report).to receive(:resource_statuses).and_return({})
    allow(fleet_client_mock).to receive(:match_profiles).and_return({ 'error' => '' })
    allow(report).to receive(:catalog_uuid).and_return(catalog_uuid)

    expect(fleet_client_mock).to receive(:match_profiles).with("#{catalog_uuid}-#{node_name}", anything)
    expect(Puppet).to receive(:info).with("Successfully matched #{node_name} with a team containing configuration profiles")

    report.process
  end

  it 'logs an error when matching profiles fails' do
    allow(report).to receive(:noop).and_return(false)
    allow(report).to receive(:resource_statuses).and_return({})
    allow(fleet_client_mock).to receive(:match_profiles).and_return({ 'error' => 'Some error' })
    allow(report).to receive(:catalog_uuid).and_return(catalog_uuid)

    expect(fleet_client_mock).to receive(:match_profiles).with("#{catalog_uuid}-#{node_name}", anything)
    expect(Puppet).to receive(:err).with("Error matching node #{node_name} with a team containing configuration profiles: Some error")

    report.process
  end
end
