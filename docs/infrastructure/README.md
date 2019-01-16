Infrastructure Documentation
============================

Kolide Fleet is an infrastructure instrumentation application which has it's own infrastructure dependencies and requirements. The infrastructure documentation contains documents on the following topics:

## Deploying and configuring osquery

- For information on installing osquery on hosts that you own, see our [Adding Hosts To Fleet](./adding-hosts-to-fleet.md) document, which compliments existing [osquery documentation](https://osquery.readthedocs.io/en/stable/).
- To add hosts to Fleet, you will need to provide a minimum set of configuration to the osquery agent on each host. These configurations are defined in the aforementioned [Adding Hosts To Fleet](./adding-hosts-to-fleet.md) document. If you'd like to further customize the osquery configurations and options, this can be done via fleetctl. You can find more documentation on this feature in the [fleetctl documentation](../cli/file-format.md#osquery-configuration-options).
- To manage osquery configurations at your organization, we strongly suggest using some form of configuration management tooling. For more information on configuration management, see the [Managing Osquery Configurations](./managing-osquery-configurations.md) document.

## Installing Fleet and its dependencies

The Fleet server has a few dependencies. To learn more about installing the Fleet server and it's dependencies, see the [Installing Fleet](./installing-fleet.md) guide.

## Managing a Fleet server

We're prepared a brief guide to help you manage and maintain your Fleet server. Check out the guide for setting up and running [Fleet on Ubuntu](./fleet-on-ubuntu.md) and [Fleet on CentOS](./fleet-on-centos.md).

For more information, you can also read the [Configuring The Fleet Binary](./configuring-the-fleet-binary.md) guide for information on how to configure and customize Fleet for your organization.

## Working with osquery logs

Fleet allows users to schedule queries, curate packs, and generate a lot of osquery logs. For more information on how you can access these logs as well as examples on what you can do with them, see the [Working With Osquery Logs](./working-with-osquery-logs.md) documentation.

## Troubleshooting & FAQ

Check out the [Frequently Asked Questions](./faq.md), which include troubleshooting steps for the most common issues experience by Fleet users.

## Security

Fleet developers have documented how Fleet handles the [OWASP Top 10](./owasp-top-10.md).
