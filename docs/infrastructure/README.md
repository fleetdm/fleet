Infrastructure Documentation
============================

Kolide Fleet is an infrastructure instrumentation application which has it's own infrastructure dependencies and requirements. The infrastructure documentation contains documents on the following topics:

## Deploying and configuring osquery

- For information on installing osquery on hosts that you own, see our [Adding Hosts To Fleet](./adding-hosts-to-fleet.md) document, which compliments existing [osquery documentation](https://osquery.readthedocs.io/en/stable/).
- To add hosts to Fleet, you will need to provide a minimum set of configuration to the osquery agent on each host. These configurations are defined in the aforementioned [Adding Hosts To Fleet](./adding-hosts-to-fleet.md) document. If you'd like to further customize the osquery configurations and options, this can be done via the Fleet application UI. You can find more documentation on this feature in the [application documentation for this feature](../application/configuring-osquery-options.md).
- To manage osquery configurations at your organization, we strongly suggest using some form of configuration management tooling. For more information on configuration management, see the [Managing Osquery Configurations](./managing-osquery-configurations.md) document.

## Installing Fleet and it's dependencies

The Fleet server has a few dependencies. To learn more about installing the Fleet server and it's dependencies, see the [Installing Fleet](./installing-fleet.md) guide.

## Managing a Fleet server

Running the Fleet server is a relatively simple process. We're prepared a brief guide to help you manage and maintain your Fleet server. Check out the guide for setting up and running [Fleet on Ubuntu](./fleet-on-ubuntu.md) and [Fleet on CentOS](./fleet-on-centos.md).

For more information, you can also read the [Configuring The Fleet Binary](./configuring-the-fleet-binary.md) guide for information on how to configure and customize Fleet for your organization.
