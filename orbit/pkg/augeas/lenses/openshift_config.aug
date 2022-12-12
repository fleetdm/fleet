(*
Module: OpenShift_Config
  Parses
    - /etc/openshift/broker.conf
    - /etc/openshift/broker-dev.conf
    - /etc/openshift/console.conf
    - /etc/openshift/console-dev.conf
    - /etc/openshift/node.conf
    - /etc/openshift/plugins.d/*.conf

Author: Brian Redbeard <redbeard@dead-city.org>

About: License
   This file is licenced under the LGPL v2+, conforming to the other components
   of Augeas.

About: Lens Usage
  Sample usage of this lens in augtool:

    * Get your current setup
      > print /files/etc/openshift
      ...

    * Change OpenShift domain
      > set /files/etc/openshift/broker.conf/CLOUD_DOMAIN ose.example.com

  Saving your file:

      > save

About: Configuration files
        /etc/openshift/broker.conf - Configuration file for an OpenShift Broker
            running in production mode.
        /etc/openshift/broker-dev.conf - Configuration file for an OpenShift
            Broker running in development mode.
        /etc/openshift/console.conf - Configuration file for an OpenShift
            console running in production mode.
        /etc/openshift/console-dev.conf - Configuration file for an OpenShift
            console running in development mode.
        /etc/openshift/node.conf - Configuration file for an OpenShift node
        /etc/openshift/plugins.d/*.conf - Configuration files for OpenShift
            plugins (i.e. mcollective configuration, remote auth, dns updates)

About: Examples
   The <Test_OpenShift_Config> file contains various examples and tests.
*)
module OpenShift_Config =
    autoload xfm

(* Variable: blank_val *)
let blank_val = del /["']{2}/ "\"\""

(* View: primary_entry *)
let primary_entry = Build.key_value_line Rx.word Sep.equal Quote.any_opt

(* View: empty_entry *)
let empty_entry = Build.key_value_line Rx.word Sep.equal blank_val

(* View: lns *)
let lns = (Util.empty | Util.comment | primary_entry | empty_entry )*

(* Variable: filter *)
let filter = incl "/etc/openshift/broker.conf"
            . incl "/etc/openshift/broker-dev.conf"
            . incl "/etc/openshift/console.conf"
            . incl "/etc/openshift/resource_limits.conf"
            . incl "/etc/openshift/console-dev.conf"
            . incl "/etc/openshift/node.conf"
            . incl "/etc/openshift/plugins.d/*.conf"
            . incl "/var/www/openshift/broker/conf/broker.conf"
            . incl "/var/www/openshift/broker/conf/plugins.d/*.conf"
            . Util.stdexcl

let xfm = transform lns filter
(* vim: set ts=4  expandtab  sw=4: *)
