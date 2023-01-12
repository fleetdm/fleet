(*
Module: UpdateDB
  Parses /etc/updatedb.conf

Author: Raphael Pinson <raphink@gmail.com>

About: Reference
  This lens tries to keep as close as possible to `man 5 updatedb.conf` where possible.

About: License
   This file is licenced under the LGPL v2+, like the rest of Augeas.

About: Lens Usage
   To be documented

About: Configuration files
   This lens applies to /etc/updatedb.conf. See <filter>.

About: Examples
   The <Test_UpdateDB> file contains various examples and tests.
*)

module UpdateDB =

autoload xfm

(* View: list
     A list entry *)
let list =
     let entry = [ label "entry" . store Rx.no_spaces ]
  in let entry_list = Build.opt_list entry Sep.space
  in [ key /PRUNE(FS|NAMES|PATHS)/ . Sep.space_equal
     . Quote.do_dquote entry_list . Util.doseol ]

(* View: bool
     A boolean entry *)
let bool = [ key "PRUNE_BIND_MOUNTS" . Sep.space_equal
           . Quote.do_dquote (store /[01]|no|yes/)
           . Util.doseol ]

(* View: lns
     The <UpdateDB> lens *)
let lns = (Util.empty|Util.comment|list|bool)*

(* Variable: filter
      The filter *)
let filter = incl "/etc/updatedb.conf"

let xfm = transform lns filter
