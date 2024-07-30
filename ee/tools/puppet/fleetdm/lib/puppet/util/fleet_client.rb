require 'net/http'
require 'uri'
require 'json'
require 'puppet'
require 'hiera_puppet'

module Puppet::Util
  # FleetClient provides an interface for making HTTP requests to a Fleet server.
  class FleetClient
    include Singleton

    # NOTE: the Puppet server supports [multithread mode][1], but it's a beta
    # feature subject to change. Still adding a mutex to control instances and
    # the cache just in case.
    #
    # [1]: https://www.puppet.com/docs/puppet/8/server/config_file_puppetserver.html
    @instance_mutex = Mutex.new

    def self.instance
      return @instance if @instance
      @instance_mutex.synchronize do
        @instance ||= new
      end
      @instance
    end

    def initialize
      @cache = {}
      @cache_mutex = Mutex.new
    end

    # Pre-assigns a profile to a host. Note that the profile assignment is not
    # effective until the sibling `match_profiles` method is called.
    #
    # @param run_identifier [String] Used to identify this run during profile matching.
    # @param uuid [String] The host uuid.
    # @param profile_xml [String] Raw XML with the configuration profile.
    # @param group [String] Used to construct a team name.
    # @param ensure_profile [String] The name of the ensure check to perform, which can be 'absent'
    # or 'present'.
    # @param environment [String] The environment (from server_facts).
    # @return [Hash] The response status code, headers, and body.
    def preassign_profile(run_identifier, uuid, profile_xml, group, ensure_profile, environment)
      req(
        method: :post,
        path: '/api/latest/fleet/mdm/apple/profiles/preassign',
        body: {
          'external_host_identifier' => run_identifier,
          'host_uuid' => uuid,
          'profile' => Base64.strict_encode64(profile_xml),
          'group' => group,
          'exclude' => ensure_profile == 'absent',
        },
        environment: environment,
      )
    end

    # Matches the set of profiles preassigned to the host (via the sibling
    # `preassign_profile` method) with a team.
    #
    # It uses `Puppet[:node_name_value]` as the `external_host_identifier`,
    # which is unique per Puppet host.
    #
    # @param run_identifier [String] Used to identify this run to match
    # pre-assigned profiles.
    # @param environment [String] The environment (from server_facts).
    # @return [Hash] The response status code, headers, and body.
    def match_profiles(run_identifier, environment)
      req(
        method: :post,
        path: '/api/latest/fleet/mdm/apple/profiles/match',
        body: { 'external_host_identifier' => run_identifier },
        environment: environment,
      )
    end

    # Sends an MDM command to the host with the specified UUID.
    #
    # @param uuid [String] The host uuid.
    # @param command_xml [String] Raw XML with the MDM command.
    # @param environment [String] The environment (from system_facts).
    # @return [Hash] The response status code, headers, and body.
    def send_mdm_command(uuid, command_xml, environment)
      req(method: :post, path: '/api/latest/fleet/mdm/apple/enqueue',
      body: {
        # For some reason, the enqueue function expects the command to be
        # base64 encoded using _raw encoding_ (without padding, as defined in RFC
        # 4648 section 3.2)
        #
        # I couldn't find a built-in Ruby function to do raw encoding, so we're
        # removing the padding manually instead.
        'command' => Base64.strict_encode64(command_xml).gsub(%r{[\n=]}, ''),
        'device_ids' => [uuid],
      },
      environment: environment)
    end

    # Get profiles assigned to the host.
    #
    # @param host_id [Number] Fleet's internal host id.
    # @param environment [String] The environment (from 'system_facts').
    # @return [Hash] The response status code, headers, and body.
    def get_host_profiles(host_id, environment)
      req(
        method: :get,
        path: "/api/latest/fleet/mdm/hosts/#{host_id}/profiles",
        cached: false,
      environment: environment,
      )
    end

    # Gets host details by host identifier.
    #
    # @param identifier [String] The host identifier, can be
    # osquery_host_identifier, node_key, UUID, or hostname.
    # @param environment [String] The environment (from server_facts).
    # @return [Hash] The response status code, headers, and body.
    def get_host_by_identifier(identifier, environment)
      req(
        method: :get,
        path: "/api/latest/fleet/hosts/identifier/#{identifier}",
        cached: true,
      environment: environment,
      )
    end

    private

    def req(method: :get, path: '', body: nil, headers: {}, cached: false, environment: 'production')
      node_name = Puppet[:node_name_value]
      node = Puppet::Node.new(node_name)
      node.environment = environment
      compiler = Puppet::Parser::Compiler.new(node)
      scope = Puppet::Parser::Scope.new(compiler)
      lookup_invocation = Puppet::Pops::Lookup::Invocation.new(scope, {}, {}, nil)
      host = Puppet::Pops::Lookup.lookup('fleetdm::host', nil, '', false, nil, lookup_invocation)
      token = Puppet::Pops::Lookup.lookup('fleetdm::token', nil, '', false, nil, lookup_invocation)

      if cached
        @cache_mutex.synchronize do
          unless @cache[path].nil?
            return @cache[path]
          end
        end
      end

      out = { 'error' => '' }
      uri = URI.parse("#{host}#{path}")
      uri.path.squeeze! '/'
      uri.path.chomp! '/'

      http = Net::HTTP.new(uri.host, uri.port)
      http.use_ssl = true if uri.scheme == 'https'

      case method
      when :get
        request = Net::HTTP::Get.new(uri.request_uri)
      when :post
        request = Net::HTTP::Post.new(uri.request_uri)
      else
        throw "HTTP method #{method} not implemented"
      end

      headers['Authorization'] = "Bearer #{token}"
      headers.each { |key, value| request[key] = value }
      request.body = body.to_json if body

      begin
        response = http.request(request)
        out = parse_response(response)

        if cached && out['error'].empty?
          @cache_mutex.synchronize do
            @cache[path] = out
          end
        end
      rescue => e
        out['error'] = e
      end

      out
    end

    def parse_response(response)
      out = {
        'status' => response.code.to_i,
        'error' => '',
        'body' => {}
      }

      if response.body
        out['body'] = JSON.parse(response.body)
      end

      if (400...600).cover?(response.code.to_i)
        message = 'server returned a non-ok status code without an error'

        if response.body
          body = JSON.parse(response.body)
          message = body['message']

          unless body['errors'].nil?
            error_messages = body['errors'].map { |e| "#{e['name']} #{e['reason']}" }
            message = [message, *error_messages].join(' : ').delete_prefix(' : ')
          end
        end

        out['error'] = message
      end

      out
    rescue JSON::ParserError => e
      {
        'status' => response.code.to_i,
       'error' => "Failed to parse response body: #{e.message}"
      }
    end
  end
end
