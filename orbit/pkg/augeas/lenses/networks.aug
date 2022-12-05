(*
Module: Networks
  Parses /etc/networks

Author: Raphael Pinson <raphink@gmail.com>

About: Reference
  This lens tries to keep as close as possible to `man 5 networks` where possible.

About: License
   This file is licenced under the LGPL v2+, like the rest of Augeas.

About: Lens Usage
   To be documented

About: Configuration files
   This lens applies to /etc/networks. See <filter>.
*)

module Networks =

autoload xfm

(* View: ipv4
    A network IP, trailing .0 may be omitted *)
let ipv4 =
  let dot     = "." in
  let digits  = /(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)/ in
    digits . (dot . digits . (dot . digits . (dot . digits)?)?)?

(*View: entry *)
let entry =
  let alias = [ seq "alias" . store Rx.word ] in
      [ seq "network" . counter "alias"
    . [ label "name" . store Rx.word ]
    . [ Sep.space . label "number" . store ipv4 ]
    . [ Sep.space . label "aliases" . Build.opt_list alias Sep.space ]?
    . (Util.eol|Util.comment) ]

(* View: lns *)
let lns = ( Util.empty | Util.comment | entry )*

(* View: filter *)
let filter = incl "/etc/networks"

let xfm = transform lns filter
