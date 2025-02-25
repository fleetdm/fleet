# Terraform Provider for FleetDM Teams

This is a Terraform provider for managing FleetDM teams. When you have
100+ teams in FleetDM, and manually managing them is not feasible. The
primary setting of concern is the team's "agent options" which
consists of some settings and command line flags. These (potentially
dangerously) configure FleetDM all machines.

## Usage

All the interesting commands are in the Makefile. If you just want
to use the thing, see `make install` and `make apply`.

Note that if you run `terraform apply` in the `tf` directory, it won't
work out of the box. That's because you need to set the 
`TF_CLI_CONFIG_FILE` environment variable to point to a file that
enables local development of this provider. The Makefile does this
for you.

Future work: actually publish this provider.

## Development

### Code Generation

See `make gen`. It will create team_resource_gen.go, which defines
the types that Terraform knows about. This is automatically run
when you run `make install`.

### Running locally

See `make plan` and `make apply`.

### Running Tests

You probably guessed this.  See `make test`. Note that these tests
require a FleetDM server to be running. The tests will create teams
and delete them when they're done. The tests also require a valid
FleetDM API token to be in the `FLEETDM_APIKEY` environment variable.

### Debugging locally

The basic idea is that you want to run the provider in a debugger.
When terraform normally runs, it will execute the provider a few
times in the course of operations. What you want to do instead is
to run the provider in debug mode and tell terraform to contact it.

To do this, you need to start the provider with the `-debug` flag
inside a debugger. You'll also need to give it the FLEETDM_APIKEY
environment variable. The provider will print out a big environment
variable that you can copy and paste to your command line.

When you run `terraform apply` or the like, you'll invoke it with
that big environment variable. It'll look something like 

```shell
TF_REATTACH_PROVIDERS='{"fleetdm.com/tf/fleetdm":{"Protocol":"grpc","ProtocolVersion":6,"Pid":33644,"Test":true,"Addr":{"Network":"unix","String":"/var/folders/32/xw2p1jtd4w10hpnsyrb_4nmm0000gq/T/plugin771405263"}}}' terraform apply
```

With this magic, terraform will look to your provider that's running
in a debugger. You get breakpoints and the goodness of a debugger.
