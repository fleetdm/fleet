(*
Module: Fuse
  Parses /etc/fuse.conf

Author: Raphael Pinson <raphink@gmail.com>

About: Reference

About: License
   This file is licenced under the LGPL v2+, like the rest of Augeas.

About: Lens Usage
   To be documented

About: Configuration files
   This lens applies to /etc/fuse.conf. See <filter>.

About: Examples
   The <Test_Fuse> file contains various examples and tests.
*)


module Fuse =
autoload xfm

(* Variable: equal *)
let equal = del /[ \t]*=[ \t]*/ " = "

(* View: mount_max *)
let mount_max = Build.key_value_line "mount_max" equal (store Rx.integer)

(* View: user_allow_other *)
let user_allow_other = Build.flag_line "user_allow_other"


(* View: lns
     The fuse.conf lens
*)
let lns = ( Util.empty | Util.comment | mount_max | user_allow_other )*

(* Variable: filter *)
let filter = incl "/etc/fuse.conf"

let xfm = transform lns filter

