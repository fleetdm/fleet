# frozen_string_literal: true

require 'spec_helper'
require_relative '../../lib/puppet/reports/fleetdm.rb'

describe 'Puppet::Util::FleetClient' do
  let(:client) { Puppet::Util::FleetClient.instance }
  let(:host) { 'https://test.example.com' }
  let(:token) { 'supersecret' }
  let(:identifier) { 'test_ident' }
  let(:rspec_puppet_env) { 'rp_env' }

  before(:each) do
    stub_const(
      'Puppet::Parser::Compiler',
      class_spy('Puppet::Parser::Compiler'),
    )

    stub_const(
      'Puppet::Parser::Scope',
      class_spy('Puppet::Parser::Scope'),
    )

    lookup = class_spy('Puppet::Pops::Lookup')
    stub_const('Puppet::Pops::Lookup', lookup)
    allow(lookup)
      .to receive(:lookup)
        .with('fleetdm::host', anything, anything, anything, anything, anything) { host }

    allow(lookup)
      .to receive(:lookup)
        .with('fleetdm::token', anything, anything, anything, anything, anything) { token }

    stub_const(
      'Puppet::Pops::Lookup::Invocation',
      class_spy('Puppet::Pops::Lookup::Invocation'),
    )
  end

  def mock_http_post(uri: '', request_body: {}, response: nil)
    mock_net_http = instance_double('Net:HTTP')
    mock_net_http_post = instance_double('Net::HTTP::POST')
    allow(Net::HTTP).to receive(:new).and_return(mock_net_http)
    allow(mock_net_http).to receive(:use_ssl=).with(true)
    allow(Net::HTTP::Post).to receive(:new).with(uri).and_return(mock_net_http_post)
    allow(mock_net_http_post).to receive(:[]=).with('Authorization', "Bearer #{token}")
    allow(mock_net_http_post).to receive(:body=).with(request_body.to_json)
    allow(mock_net_http).to receive(:request).with(mock_net_http_post) { response }
  end

  def mock_http_get(uri: '', response: instance_double(Net::HTTPSuccess, code: 204, body: nil))
    mock_net_http = instance_double('Net:HTTP')
    mock_net_http_get = instance_double('Net::HTTP::POST')
    allow(Net::HTTP).to receive(:new).and_return(mock_net_http)
    allow(mock_net_http).to receive(:use_ssl=).with(true)
    allow(Net::HTTP::Get).to receive(:new).with(uri).and_return(mock_net_http_get)
    allow(mock_net_http_get).to receive(:[]=).with('Authorization', "Bearer #{token}")
    allow(mock_net_http).to receive(:request).with(mock_net_http_get) { response }
  end

  describe '#match_profiles' do
    describe 'successful response' do
      subject :result do
        mock_http_post(
          uri: '/api/latest/fleet/mdm/apple/profiles/match',
          request_body: { 'external_host_identifier' => identifier },
          response: instance_double(Net::HTTPSuccess, code: 204, body: nil),
        )
        client.match_profiles(identifier, rspec_puppet_env)
      end

      it { expect(result['body']).to eq({}) }
      it { expect(result['error']).to eq('') }
      it { expect(result['status']).to eq(204) }
    end

    describe 'response with errors' do
      subject :result do
        mock_http_post(
          uri: '/api/latest/fleet/mdm/apple/profiles/match',
          request_body: { 'external_host_identifier' => identifier },
          response: instance_double(
            Net::HTTPServerError,
            code: 500,
            body: body.to_json,
          ),
        )
        client.match_profiles(identifier, rspec_puppet_env)
      end

      let(:body) do
        { 'errors' => [{ 'name' => 'server error', 'reason' => 'unknown' }] }
      end

      it { expect(result['body']).to eq(body) }
      it { expect(result['error']).to eq('server error unknown') }
      it { expect(result['status']).to eq(500) }
    end
  end
end
