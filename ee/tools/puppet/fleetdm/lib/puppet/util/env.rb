require 'puppet'
require 'hiera_puppet'

module Puppet::Util
  def read_hiera(key)
    node_name = Puppet[:node_name_value]
    node = Puppet::Node.new(node_name)
    node.environment = Puppet.lookup(:current_environment).name.to_s
    compiler = Puppet::Parser::Compiler.new(node)
    scope = Puppet::Parser::Scope.new(compiler)
    lookup_invocation = Puppet::Pops::Lookup::Invocation.new(scope, {}, {}, nil)
    Puppet::Pops::Lookup.lookup(key, nil, '', false, nil, lookup_invocation)
  end
end
