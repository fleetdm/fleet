require 'net/http'
require 'uri'
require 'json'
require 'puppet'
require 'hiera_puppet'

module Puppet::Util
  # FleetClient provides an interface for making HTTP requests to a Fleet server.
  class FleetClient
    def initialize(host, token)
      @host = host
      @token = token
    end

    # Pre-assigns a profile to a host. Note that the profile assignment is not
    # effective until the sibling `match_profiles` method is called.
    #
    # @param uuid [String] The host uuid.
    # @param profile_xml [String] Raw XML with the configuration profile.
    # @param group [String] Used to construct a team name.
    # @return [Hash] The response status code, headers, and body.
    def preassign_profile(uuid, profile_xml, group)
      post(
        '/api/latest/fleet/mdm/apple/profiles/preassign',
        {
          'external_host_identifier' => Puppet[:node_name_value],
          'host_uuid' => uuid,
          'profile' => Base64.encode64(profile_xml),
          'group' => group,
        },
      )
    end

    # Matches the set of profiles preassigned to the host (via the sibling
    # `preassign_profile` method) with a team.
    #
    # It uses `Puppet[:node_name_value]` as the `external_host_identifier`,
    # which is unique per Puppet host.
    #
    # @return [Hash] The response status code, headers, and body.
    def match_profiles
      post('/api/latest/fleet/mdm/apple/profiles/match',
  {
    'external_host_identifier' => Puppet[:node_name_value],
  })
    end

    # Sends an MDM command to the host with the specified UUID.
    #
    # @param uuid [String] The host uuid.
    # @param command_xml [String] Raw XML with the MDM command.
    # @return [Hash] The response status code, headers, and body.
    def send_mdm_command(uuid, command_xml)
      post('/api/latest/fleet/mdm/apple/enqueue',
      {
        'command' => command_xml,
        'device_ids' => [uuid],
      })
    end

    # Sends an HTTP POST request to the specified path.
    #
    # @param path [String] The path of the resource to post to.
    # @param body [Object] (optional) The request body to send.
    # @param headers [Hash] (optional) Additional headers to include in the request.
    # @return [Hash] The response status code, headers, and body.
    def post(path, body = nil, headers = {})
      uri = URI.parse("#{@host}#{path}")

      http = Net::HTTP.new(uri.host, uri.port)
      http.use_ssl = true if uri.scheme == 'https'

      request = Net::HTTP::Post.new(uri.request_uri)

      headers['Authorization'] = "Bearer #{@token}"
      headers.each { |key, value| request[key] = value }
      request.body = body.to_json if body

      response = http.request(request)
      parse_response(response)
    end

    private

    def parse_response(response)
      {
        status: response.code.to_i,
        headers: response.to_hash,
        body: response.body ? JSON.parse(response.body) : nil,
      }
    rescue JSON::ParserError => e
      {
        status: response.code.to_i,
        headers: response.to_hash,
        error: "Failed to parse response body: #{e.message}"
      }
    end
  end
end
