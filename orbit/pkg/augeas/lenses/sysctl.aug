(*
Module: Sysctl
  Parses /etc/sysctl.conf and /etc/sysctl.d/*

Author: David Lutterkort <lutter@redhat.com>

About: Reference

About: License
   This file is licenced under the LGPL v2+, like the rest of Augeas.

About: Lens Usage
   To be documented

About: Configuration files
   This lens applies to /etc/sysctl.conf and /etc/sysctl.d/*. See <filter>.

About: Examples
   The <Test_Sysctl> file contains various examples and tests.
*)

module Sysctl =
autoload xfm

(* Variable: filter *)
let filter = incl "/boot/loader.conf"
           . incl "/etc/sysctl.conf"
           . incl "/etc/sysctl.d/*"
           . excl "/etc/sysctl.d/README"
           . excl "/etc/sysctl.d/README.sysctl"
           . Util.stdexcl

(* View: comment *)
let comment = Util.comment_generic /[ \t]*[#;][ \t]*/ "# "

(* View: entry
   basically a Simplevars.entry but key has to allow some special chars as '*' *)
let entry =
     let some_value = Sep.space_equal . store Simplevars.to_comment_re
  (* Rx.word extended by * and : *)
  in let word = /[*:\/A-Za-z0-9_.-]+/
  (* Avoid ambiguity in tree by making a subtree here *)
  in let empty_value = [del /[ \t]*=/ "="] . store ""
  in [ Util.indent . key word
            . (some_value? | empty_value)
            . (Util.eol | Util.comment_eol) ]

(* View: lns
     The sysctl lens *)
let lns = (Util.empty | comment | entry)*

let xfm = transform lns filter
