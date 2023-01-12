(*
Module: Postfix_Transport
  Parses /etc/postfix/transport

Author: Raphael Pinson <raphael.pinson@camptocamp.com>

About: Reference
  This lens tries to keep as close as possible to `man 5 transport` where possible.

About: License
   This file is licenced under the LGPL v2+, like the rest of Augeas.

About: Lens Usage
   To be documented

About: Configuration files
   This lens applies to /etc/postfix/transport. See <filter>.

About: Examples
   The <Test_Postfix_Transport> file contains various examples and tests.
*)

module Postfix_Transport =

autoload xfm

(* View: space_or_eol *)
let space_or_eol = del /([ \t]*\n)?[ \t]+/ " "

(* View: colon *)
let colon = Sep.colon

(* View: nexthop *)
let nexthop =
     let host_re = "[" . Rx.word . "]" | /[A-Za-z]([^\n]*[^ \t\n])?/
  in [ label "nexthop" . (store host_re)? ]

(* View: transport *)
let transport = [ label "transport" . (store Rx.word)? ]
                . colon . nexthop

(* View: nexthop_smtp *)
let nexthop_smtp =
     let host_re = "[" . Rx.word . "]" | Rx.word
  in [ label "host" . store host_re ]
     . colon
     . [ label "port" . store Rx.integer ]

(* View: record *)
let record = [ label "pattern" . store /[A-Za-z0-9@\*._-]+/
             . space_or_eol . (transport | nexthop_smtp)
             . Util.eol ]

(* View: lns *)
let lns = (Util.empty | Util.comment | record)*

(* Variable: filter *)
let filter = incl "/etc/postfix/transport"
           . incl "/usr/local/etc/postfix/transport"

let xfm = transform lns filter
