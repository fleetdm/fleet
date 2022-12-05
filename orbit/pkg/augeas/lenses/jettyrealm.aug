(*
Module: JettyRealm
  JettyRealm Properties for Augeas

Author: Brian Redbeard <redbeard@dead-city.org>

About: Reference
  This lens ensures that properties files for JettyRealms are properly
  handled by Augeas.

About: License
  This file is licensed under the LGPL License.

About: Lens Usage
  Sample usage of this lens in augtool:

    * Create a new user
      > ins user  after /files/etc/activemq/jetty-realm.properties/user
      > set /files/etc/activemq/jetty-realm.properties/user[last()]/username redbeard
      > set /files/etc/activemq/jetty-realm.properties/user[last()]/password testing
      > set /files/etc/activemq/jetty-realm.properties/user[last()]/realm admin
      ...

    * Delete the user named sample_user
      > rm /files/etc/activemq/jetty-realm.properties/user[*][username = "sample_user"]

  Saving your file:

      > save

About: Configuration files
  This lens applies to jetty-realm.properties files. See <filter>.
*)

module JettyRealm =
        autoload xfm


(* View: comma_sep *)
let comma_sep = del /,[ \t]*/ ", "

(* View: realm_entry *)
let realm_entry = [ label "user" .
                    [ label "username" . store Rx.word ] . del /[ \t]*:[ \t]*/ ": " .
                    [ label "password" . store Rx.word ] . 
                    [ label "realm" . comma_sep . store Rx.word ]* .
                    Util.eol ]

(* View: lns *)
let lns = ( Util.comment | Util.empty | realm_entry )*


(* Variable: filter *)
let filter = incl "/etc/activemq/jetty-realm.properties"

let xfm = transform lns filter
