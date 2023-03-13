# frozen_string_literal: true

require 'net/http'
require 'net/https'
require 'uri'
require 'json'

module Puppet
  module Util
    class FleetClient
      def initialize(host, token)
        @host = host
        @token = token
      end

      def transfer_host(_team_id, host_uuid)
        uri = URI.parse("#{@host}/api/v1/fleet/hosts/transfer/filter")
        req = Net::HTTP::Post.new(uri.request_uri)
        # TODO(roperzh): last minute I refactored this into a module and
        # the team_id is coming as nil, figure out why and adjust instead
        # of hardcoding.
        data = {
          'filters' => { query: host_uuid },
          'team_id' => 1
        }
        req.body = data.to_json
        send(uri, req)
      end

      def team_id_from_name(team_name)
        uri = URI.parse("#{@host}/api/v1/fleet/teams?query=#{team_name}")
        req = Net::HTTP::Get.new(uri.request_uri)
        send(uri, req)
      end

      def batch_send_profiles(team_name, profiles)
        uri = URI.parse("#{@host}/api/latest/fleet/mdm/apple/profiles/batch?team_name=#{team_name}")
        req = Net::HTTP::Post.new(uri.request_uri)
        data = { 'profiles' => profiles }
        req.body = data.to_json
        send(uri, req)
      end

      def send(uri, req)
        output = {}
        output['error'] = false
        output['error_message'] = ''
        http = Net::HTTP.new(uri.host, uri.port)
        http.use_ssl = true
        req['Authorization'] = "Bearer #{@token}"

        begin
          response = http.request(req)
        rescue StandardError => e
          output['error'] = true
          output['error_message'] = e
        end

        if response.is_a?(Net::HTTPSuccess) || response.is_a?(Net::HTTPNoContent)
          output['output'] = response.body unless response.body.nil?
        else
          output['error'] = true
          output['error_message'] = response.code
        end

        output
      end
    end
  end
end
