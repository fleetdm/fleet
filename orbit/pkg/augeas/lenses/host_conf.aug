(*
Module: Host_Conf
  Parses /etc/host.conf

Author: Raphael Pinson <raphink@gmail.com>

About: Reference
  This lens tries to keep as close as possible to `man 5 host.conf` where possible.

About: License
   This file is licenced under the LGPL v2+, like the rest of Augeas.

About: Lens Usage
   To be documented

About: Configuration files
   This lens applies to /etc/host.conf. See <filter>.
*)

module Host_Conf =

autoload xfm

(************************************************************************
 * Group:                 ENTRY TYPES
 *************************************************************************)

(* View: sto_bool
    Store a boolean value *)
let sto_bool = store ("on"|"off")

(* View: sto_bool_warn
    Store a boolean value *)
let sto_bool_warn = store ("on"|"off"|"warn"|"nowarn")

(* View: bool
    A boolean switch *)
let bool (kw:regexp) = Build.key_value_line kw Sep.space sto_bool

(* View: bool_warn
    A boolean switch with extended values *)
let bool_warn (kw:regexp) = Build.key_value_line kw Sep.space sto_bool_warn

(* View: list
    A list of items *)
let list (kw:regexp) (elem:string) =
  let list_elems = Build.opt_list [seq elem . store Rx.word] (Sep.comma . Sep.opt_space) in
  Build.key_value_line kw Sep.space list_elems

(* View: trim *)
let trim =
  let trim_list = Build.opt_list [seq "trim" . store Rx.word] (del /[:;,]/ ":") in
  Build.key_value_line "trim" Sep.space trim_list

(* View: entry *)
let entry = bool ("multi"|"nospoof"|"spoofalert"|"reorder")
          | bool_warn "spoof"
          | list "order" "order"
          | trim

(************************************************************************
 * Group:                 LENS AND FILTER
 *************************************************************************)

(* View: lns *)
let lns = ( Util.empty | Util.comment | entry )*

(* View: filter *)
let filter = incl "/etc/host.conf"

let xfm = transform lns filter
