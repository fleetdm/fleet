(*
Module: Cmdline
  Parses /proc/cmdline and /etc/kernel/cmdline

Author: Thomas Weißschuh <thomas.weissschuh@amadeus.com>

About: License
  This file is licensed under the LGPL v2+, like the rest of Augeas.
*)

module Cmdline =
  autoload xfm

let entry = [ key Rx.word . Util.del_str "=" . store Rx.no_spaces ] | [ key Rx.word ]

let lns = (Build.opt_list entry Sep.space)? . del /\n?/ ""

let filter = incl "/etc/kernel/cmdline"
           . incl "/proc/cmdline"

let xfm = transform lns filter
