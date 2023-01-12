(*
Module: JMXPassword
  JMXPassword for Augeas

Author: Brian Redbeard <redbeard@dead-city.org>


About: Reference
  This lens ensures that files included in JMXPassword are properly
  handled by Augeas.

About: License
  This file is licensed under the LGPL License.

About: Lens Usage
  Sample usage of this lens in augtool:

    * Create a new user
      > ins user  after /files/etc/activemq/jmx.password
      > set /files/etc/activemq/jmx.password/user[last()]/username redbeard
      > set /files/etc/activemq/jmx.password/user[last()]/password testing
      ...

    * Delete the user named sample_user
      > rm /files/etc/activemq/jmx.password/user[*][username = "sample_user"]

  Saving your file:

      > save

About: Configuration files
  This lens applies to relevant conf files located in  /etc/activemq/ 
  The following views correspond to the related files:
    * pass_entry:
      /etc/activemq/jmx.password
  See <filter>.
  

*)

module JMXPassword =
        autoload xfm

(* View: pass_entry *)
let pass_entry = [ label "user" .
                    [ label "username" . store Rx.word ] . Sep.space .
                    [ label "password" . store Rx.no_spaces ] . Util.eol ]

(* View: lns *)
let lns = ( Util.comment | Util.empty | pass_entry )*


(* Variable: filter *)
let filter = incl "/etc/activemq/jmx.password"

let xfm = transform lns filter
