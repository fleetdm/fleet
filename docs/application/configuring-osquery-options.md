Configuring Osquery Options
===========================

To connect a host to Fleet, you have to launch `osqueryd` with some very specific options (which are further outlined in the [Adding Hosts To Fleet](../infrastructure/adding-hosts-to-fleet.md) documentation). Once a host has connected, many global osquery configuration options can be changed without necessitating a reboot of osquery. These options can be edited and managed by selecting the "Config" sidebar.

![Manage Osquery Options](../images/manage-osquery-options.png)

Note that this is a more advanced feature. The Fleet application will not allow you to configure osquery such that it can't talk to Fleet anymore, but you can significantly modify the behavior of osquery via configuration.
