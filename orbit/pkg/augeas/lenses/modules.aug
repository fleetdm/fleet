(*
Module: Modules
  Parses /etc/modules

About: Reference
  This lens tries to keep as close as possible to `man 5 modules` where possible.

About: License
   This file is licenced under the LGPL v2+, like the rest of Augeas.

About: Lens Usage
   To be documented

About: Configuration files
   This lens applies to /etc/modules. See <filter>.
*)
module Modules =
autoload xfm

(* View: word *)
let word = /[^#, \n\t\/]+/

(* View: sto_line *)
let sto_line = store /[^# \t\n].*[^ \t\n]|[^# \t\n]/

(* View: record *)
let record = [ key word . (Util.del_ws_tab . sto_line)? . Util.eol ]

(* View: lns *)
let lns = ( Util.empty | Util.comment | record ) *

(* View: filter *)
let filter = incl "/etc/modules"

let xfm = transform lns filter
