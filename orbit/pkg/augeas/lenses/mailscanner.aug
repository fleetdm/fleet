(*
Module: Mailscanner
  Parses MailScanner configuration files.

Author: Andrew Colin Kissa <andrew@topdog.za.net>
  Baruwa Enterprise Edition http://www.baruwa.com

About: License
  This file is licensed under the LGPL v2+.

About: Configuration files
  This lens applies to /etc/MailScanner/MailScanner.conf and files in 
  /etc/MailScanner/conf.d/. See <filter>.
*)

module Mailscanner =
autoload xfm

(************************************************************************
 * Group: USEFUL PRIMITIVES
 *************************************************************************)
let comment  = Util.comment

let empty = Util.empty

let space = Sep.space

let eol = Util.eol

let non_eq = /[^ =\t\r\n]+/

let non_space = /[^# \t\n]/

let any = /.*/

let word = /[A-Za-z%][ :<>%A-Za-z0-9_.-]+[A-Za-z%2]/

let include_kw = /include/

let keys = word - include_kw

let eq         = del /[ \t]*=/ " ="

let indent     = del /[ \t]*(\n[ \t]+)?/ " "

let line_value = store (non_space . any .  non_space | non_space)

(************************************************************************
 * Group: Entries 
 *************************************************************************)

let include_line = Build.key_value_line include_kw space (store non_eq)

let normal_line = [ key keys . eq . (indent . line_value)? . eol ]

(************************************************************************
 * Group: Lns and Filter
 *************************************************************************)

let lns = (empty|include_line|normal_line|comment) *

let filter = (incl "/etc/MailScanner/MailScanner.conf")
            . (incl "/etc/MailScanner/conf.d/*.conf")

let xfm = transform lns filter
