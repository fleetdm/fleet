(*
Module: Pgbouncer
 Parses Pgbouncer ini configuration files.

Author: Andrew Colin Kissa <andrew@topdog.za.net>
 Baruwa Enterprise Edition http://www.baruwa.com

About: License
 This file is licensed under the LGPL v2+.

About: Configuration files
 This lens applies to /etc/pgbouncer.ini See <filter>.

About: TODO
 Create a tree for the database options
*)

module Pgbouncer =
autoload xfm

(************************************************************************
 * Group:                 USEFUL PRIMITIVES
 ************************************************************************)

let comment  = IniFile.comment IniFile.comment_re IniFile.comment_default

let sep = IniFile.sep "=" "="

let eol = Util.eol

let entry_re = ( /[A-Za-z][:#A-Za-z0-9._-]+|\*/)

(************************************************************************
 * Group:                       ENTRY
 *************************************************************************)

let non_db_line = [ key entry_re . sep . IniFile.sto_to_eol? . eol ]

let entry = non_db_line|comment

let title   = IniFile.title IniFile.record_re

let record  = IniFile.record title entry

(******************************************************************
 * Group:                   LENS AND FILTER
 ******************************************************************)

let lns = IniFile.lns record comment

(* Variable: filter *)
let filter = incl "/etc/pgbouncer.ini"

let xfm = transform lns filter

