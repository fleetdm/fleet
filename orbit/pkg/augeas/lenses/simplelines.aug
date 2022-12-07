(*
Module: Simplelines
   Parses simple lines conffiles

Author: Raphael Pinson <raphink@gmail.com>

About: License
   This file is licenced under the LGPL v2+, like the rest of Augeas.

About: Lens Usage
   To be documented

About: Configuration files
   See <filter>.

About: Examples
   The <Test_Simplelines> file contains various examples and tests.
*)

module Simplelines =

autoload xfm

(* View: line
     A simple, uncommented, line *)
let line =
   let line_re = /[^# \t\n].*[^ \t\n]|[^# \t\n]/
   in [ seq "line" . Util.indent
      . store line_re . Util.eol ]

(* View: lns
     The simplelines lens *)
let lns = (Util.empty | Util.comment | line)*

(* Variable: filter *)
let filter = incl "/etc/at.allow"
           . incl "/etc/at.deny"
           . incl "/etc/cron.allow"
           . incl "/etc/cron.deny"
           . incl "/etc/cron.d/at.allow"
           . incl "/etc/cron.d/at.deny"
           . incl "/etc/cron.d/cron.allow"
           . incl "/etc/cron.d/cron.deny"
           . incl "/etc/default/grub_installdevice"
           . incl "/etc/pam.d/allow.pamlist"
           . incl "/etc/hostname.*"

let xfm = transform lns filter
