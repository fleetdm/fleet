(*
Module: BackupPCHosts
  Parses /etc/backuppc/hosts

About: Reference
  This lens tries to keep as close as possible to `man backuppc` where possible.

About: License
   This file is licenced under the LGPL v2+, like the rest of Augeas.

About: Lens Usage
   To be documented

About: Configuration files
   This lens applies to /etc/backuppc/hosts. See <filter>.
*)

module BackupPCHosts =
autoload xfm

(* View: word *)
let word = /[^#, \n\t\/]+/

(* View: record *)
let record =
  let moreusers = Build.opt_list [ label "moreusers" . store word ] Sep.comma in
              [ seq "host"
              . [ label "host" . store word ] . Util.del_ws_tab
              . [ label "dhcp" . store word ] . Util.del_ws_tab
              . [ label "user" . store word ]
              . (Util.del_ws_tab . moreusers)?
              . (Util.comment|Util.eol) ]

(* View: lns *)
let lns = ( Util.empty | Util.comment | record ) *

(* View: filter *)
let filter = incl "/etc/backuppc/hosts"

let xfm = transform lns filter
