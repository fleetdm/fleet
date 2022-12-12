(*
Module: CPanel
  Parses cpanel.config

Author: Raphael Pinson <raphink@gmail.com>

About: Reference
  This lens parses cpanel.config files

About: License
   This file is licenced under the LGPL v2+, like the rest of Augeas.

About: Lens Usage
   To be documented

About: Configuration files
   This lens applies to cpanel.config files. See <filter>.

About: Examples
   The <Test_CPanel> file contains various examples and tests.
*)
module CPanel =

autoload xfm

(* View: kv
    A key-value pair, supporting flags and empty values *)
let kv = [ key /[A-Za-z0-9:_.-]+/
         . (Sep.equal . store (Rx.space_in?))?
         . Util.eol ]

(* View: lns
    The <CPanel> lens *)
let lns = (Util.comment | Util.empty | kv)* 

(* View: filter *)
let filter = incl "/var/cpanel/cpanel.config"

let xfm = transform lns filter
