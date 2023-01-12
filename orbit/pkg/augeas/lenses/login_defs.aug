(*
Module: Login_defs
  Lense for login.defs

Author: Erinn Looney-Triggs

About: License
  This file is licensed under the LGPL v2+, like the rest of Augeas.

About: Configuration files
   This lens applies to /etc/login.defs. See <filter>.
*)
module Login_defs =
autoload xfm

(* View: record
    A login.defs record *)
let record =
  let value = store /[^ \t\n]+([ \t]+[^ \t\n]+)*/ in
  [ key Rx.word . Sep.space . value . Util.eol ]

(* View: lns
    The login.defs lens *)
let lns = (record | Util.comment | Util.empty) *

(* View: filter *)
let filter = incl "/etc/login.defs"

let xfm = transform lns filter
