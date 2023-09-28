(*
Module: Simplevars
  Parses simple key = value conffiles

Author: Raphael Pinson <raphink@gmail.com>

About: License
   This file is licenced under the LGPL v2+, like the rest of Augeas.

About: Lens Usage
   To be documented

About: Examples
   The <Test_Simplevars> file contains various examples and tests.
*)

module Simplevars =

autoload xfm

(* Variable: to_comment_re
   The regexp to match the value *)
let to_comment_re =
     let to_comment_squote = /'[^\n']*'/
  in let to_comment_dquote = /"[^\n"]*"/
  in let to_comment_noquote = /[^\n \t'"#][^\n#]*[^\n \t#]|[^\n \t'"#]/
  in to_comment_squote | to_comment_dquote | to_comment_noquote

(* View: entry *)
let entry =
     let some_value = Sep.space_equal . store to_comment_re
     (* Avoid ambiguity in tree by making a subtree here *)
  in let empty_value = [del /[ \t]*=/ "="] . store ""
  in [ Util.indent . key Rx.word
            . (some_value? | empty_value)
            . (Util.eol | Util.comment_eol) ]

(* View: lns *)
let lns = (Util.empty | Util.comment | entry)*

(* Variable: filter *)
let filter = incl "/etc/kernel-img.conf"
           . incl "/etc/kerneloops.conf"
           . incl "/etc/wgetrc"
           . incl "/etc/zabbix/*.conf"
           . incl "/etc/zabbix/*/*.conf"
           . incl "/etc/audit/auditd.conf"
           . incl "/etc/mixerctl.conf"
           . incl "/etc/wsconsctlctl.conf"
           . incl "/etc/ocsinventory/ocsinventory-agent.cfg"

let xfm = transform lns filter
