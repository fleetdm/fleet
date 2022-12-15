(*
Module: ActiveMQ_Conf
  ActiveMQ / FuseMQ conf module for Augeas

Author: Brian Redbeard <redbeard@dead-city.org>

About: Reference
  This lens ensures that conf files included in ActiveMQ /FuseMQ are properly
  handled by Augeas.

About: License
  This file is licensed under the LGPL License.

About: Lens Usage
  Sample usage of this lens in augtool:

    * Get your current setup
      > print /files/etc/activemq.conf
      ...

    * Change ActiveMQ Home
      > set /files/etc/activemq.conf/ACTIVEMQ_HOME /usr/share/activemq

  Saving your file:

      > save

About: Configuration files
  This lens applies to relevant conf files located in  /etc/activemq/ and 
  the file /etc/activemq.conf . See <filter>.

*)

module ActiveMQ_Conf =
        autoload xfm

(* Variable: blank_val *)
let blank_val = del /^\z/

(* View: entry *)
let entry =
  Build.key_value_line Rx.word Sep.space_equal Quote.any_opt

(* View: empty_entry *)
let empty_entry = Build.key_value_line Rx.word Sep.equal  Quote.dquote_opt_nil

(* View: lns *)
let lns = (Util.empty | Util.comment | entry | empty_entry )*

(* Variable: filter *)
let filter = incl "/etc/activemq.conf"
           . incl "/etc/activemq/*"
           . excl "/etc/activemq/*.xml"
           . excl "/etc/activemq/jmx.*"
           . excl "/etc/activemq/jetty-realm.properties"
           . excl "/etc/activemq/*.ts"
           . excl "/etc/activemq/*.ks"
           . excl "/etc/activemq/*.cert"
           . Util.stdexcl

let xfm = transform lns filter
