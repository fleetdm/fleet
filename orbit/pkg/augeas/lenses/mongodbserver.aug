(*
Module: MongoDBServer
  Parses /etc/mongodb.conf

Author: Brian Redbeard <redbeard@dead-city.org>

About: Reference
   For information on configuration options available to mongod reference one
   of the following resources:
    * The Mongo DB Manual - <http://docs.mongodb.org/manual/>
    * The current options available for your operating system via:
      > man mongos

About: License
   This file is licenced under the LGPL v2+, conforming to the other components
   of Augeas.

About: Lens Usage
  Sample usage of this lens in augtool:

    * Get your current setup
      > print /files/etc/mongodb.conf
      ...

    * Change MongoDB port
      > set /files/etc/mongodb.conf/port 27117

  Saving your file:

      > save

About: Configuration files
   This lens applies to /etc/mongodb.conf. See <filter>.

About: Examples
   The <Test_MongoDBServer> file contains various examples and tests.
*)
module MongoDBServer =

autoload xfm

(* View: entry *)
let entry =
  Build.key_value_line Rx.word Sep.space_equal (store Rx.space_in)

(* View: lns *)
let lns = (Util.empty | Util.comment | entry)*


(* Variable: filter *)
let filter = incl "/etc/mongodb.conf"

let xfm = transform lns filter
