(*
Module: JMXAccess
  JMXAccess module for Augeas

Author: Brian Redbeard <redbeard@dead-city.org>


About: Reference
  This lens ensures that files included in JMXAccess are properly
  handled by Augeas.

About: License
  This file is licensed under the LGPL License.

About: Lens Usage
  Sample usage of this lens in augtool:

    * Create a new user
      > ins user  after /files/etc/activemq/jmx.access
      > set /files/etc/activemq/jmx.password/user[last()]/username redbeard
      > set /files/etc/activemq/jmx.password/user[last()]/access readonly
      ...

    * Delete the user named sample_user
      > rm /files/etc/activemq/jmx.password/user[*][username = "sample_user"]

  Saving your file:

      > save

About: Configuration files
  This lens applies to relevant conf files located in  /etc/activemq/ 
  The following views correspond to the related files:
    * access_entry:
      /etc/activemq/jmx.access
  See <filter>.
  

*)

module JMXAccess =
        autoload xfm

(* View: access_entry *)
let access_entry = [ label "user" .
                    [ label "username" . store Rx.word ] . Sep.space .
                    [ label "access" . store /(readonly|readwrite)/i ] . Util.eol ]



(* View: lns *)
let lns = ( Util.comment | Util.empty | access_entry )*


(* Variable: filter *)
let filter = incl "/etc/activemq/jmx.access"

let xfm = transform lns filter
