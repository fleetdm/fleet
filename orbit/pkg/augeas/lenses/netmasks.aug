(*
Module: Netmasks
  Parses /etc/inet/netmasks on Solaris

Author: Dominic Cleal <dcleal@redhat.com>

About: Reference
  This lens tries to keep as close as possible to `man 4 netmasks` where possible.

About: Licence
  This file is licensed under the LGPL v2+, like the rest of Augeas.

About: Lens Usage

About: Configuration files
  This lens applies to /etc/netmasks and /etc/inet/netmasks. See <filter>.
*)

module Netmasks =
autoload xfm

(************************************************************************
 * Group:                 USEFUL PRIMITIVES
 ************************************************************************)

(* View: comment *)
let comment = Util.comment

(* View: comment_or_eol *)
let comment_or_eol = Util.comment_or_eol

(* View: indent *)
let indent  = Util.indent

(* View: empty *)
let empty   = Util.empty

(* View: sep
    The separator for network/mask entries *)
let sep     = Util.del_ws_tab

(************************************************************************
 * Group:                     ENTRIES
 ************************************************************************)

(* View: entry
   Network / netmask line *)
let entry = [ seq "network" . indent .
                [ label "network" . store Rx.ipv4 ] . sep .
                [ label "netmask" . store Rx.ipv4 ] . comment_or_eol ]

(************************************************************************
 * Group:                     LENS
 ************************************************************************)

(* View: lns *)
let lns = ( empty
          | comment
          | entry )*

(* Variable: filter *)
let filter = (incl "/etc/netmasks"
            . incl "/etc/inet/netmasks")

let xfm = transform lns filter
