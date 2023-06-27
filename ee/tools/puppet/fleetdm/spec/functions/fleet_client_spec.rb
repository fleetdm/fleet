# frozen_string_literal: true

require 'spec_helper'

describe 'Puppet::Util::FleetClient' do
  let(:client) { Puppet::Util::FleetClient.new('https://example.com', 'token') }

  it 'handles POST with 204 responses' do
    response = Net::HTTPSuccess.new(1.0, '204', 'OK')
    expect_any_instance_of(Net::HTTP).to receive(:request) { response } # rubocop:disable RSpec/AnyInstance

    result = client.post('/example')
    expect(result[:body]).to be(nil)
  end
end
