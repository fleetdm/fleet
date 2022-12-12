(*
Module: Protocols
  Parses /etc/protocols

Author: Raphael Pinson <raphink@gmail.com>

About: Reference
  This lens tries to keep as close as possible to `man 5 protocols` where possible.

About: License
   This file is licenced under the LGPL v2+, like the rest of Augeas.

About: Lens Usage
   To be documented

About: Configuration files
   This lens applies to /etc/protocols. See <filter>.

About: Examples
   The <Test_Protocols> file contains various examples and tests.
*)


module Protocols =

autoload xfm

let protoname = /[^# \t\n]+/

(* View: entry *)
let entry =
      let protocol = [ label "protocol" . store protoname ]
   in let number   = [ label "number" . store Rx.integer ]
   in let alias    = [ label "alias" . store protoname ]
   in [ seq "protocol" . protocol
      . Sep.space . number
      . (Sep.space . Build.opt_list alias Sep.space)?
      . Util.comment_or_eol ]

(* View: lns
     The protocols lens *)
let lns = (Util.empty | Util.comment | entry)*

(* Variable: filter *)
let filter = incl "/etc/protocols"

let xfm = transform lns filter
