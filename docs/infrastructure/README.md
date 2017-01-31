Infrastructure Documentation
============================

Kolide is an infrastructure instrumentation application which has it's own infrastructure dependencies and requirements. The infrastructure documentation contains documents on the following topics:

## Deploying and configuring osquery

- For information on installing osquery on hosts that you own, see our [Adding Hosts To Kolide](./adding-hosts-to-kolide.md) document, which compliments existing [osquery documentation](https://osquery.readthedocs.io/en/stable/).
- To add hosts to Kolide, you will need to provide a minimum set of configuration to the osquery agent on each host. These configurations are defined in the aforementioned [Adding Hosts To Kolide](./adding-hosts-to-kolide.md) document. If you'd like to further customize the osquery configurations and options, this can be done via the Kolide application UI. You can find more documentation on this feature in the [application documentation for this feature](../application/configuring-osquery-options.md).
- To manage osquery configurations at your organization, we strongly suggest using some form of configuration management tooling. For more information on configuration management, see the [Managing Osquery Configurations](./managing-osquery-configurations.md) document.

## Installing Kolide and it's dependencies

The Kolide server has a few dependencies. To learn more about installing the Kolide server and it's dependencies, see the [Installing Kolide](./installing-kolide.md) guide.

## Managing a Kolide server

Running the Kolide server is a relatively simple process. We're prepared a brief guide to help you manage and maintain your Kolide server. Check out the guide for setting up and running [Kolide on Ubuntu](./kolide-on-ubuntu.md) and [Kolide on CentOS](./kolide-on-centos.md).

For more information, you can also read the [Configuring The Kolide Binary](./configuring-the-kolide-binary.md) guide for information on how to configure and customize Kolide for your organization.
