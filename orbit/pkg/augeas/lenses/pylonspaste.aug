(*
Module: PylonsPaste
 Parses Pylons Paste ini configuration files.

Author: Andrew Colin Kissa <andrew@topdog.za.net>
 Baruwa Enterprise Edition http://www.baruwa.com

About: License
 This file is licensed under the LGPL v2+.

About: Configuration files
 This lens applies to /etc/baruwa See <filter>.
*)

module Pylonspaste =
autoload xfm

(************************************************************************
 * Group: USEFUL PRIMITIVES
 *************************************************************************)

let comment  = IniFile.comment IniFile.comment_re IniFile.comment_default

let sep = IniFile.sep "=" "="

let eol = Util.eol

let optspace = del /\n/ "\n"

let entry_re = ( /[A-Za-z][:#A-Za-z0-9._-]+/)

let plugin_re = /[A-Za-z][;:#A-Za-z0-9._-]+/

let plugins_kw = /plugins/

let debug_kw = /debug/

let normal_opts = entry_re - (debug_kw|plugins_kw)

let del_opt_ws =  del /[\t ]*/ ""

let new_ln_sep = optspace . del_opt_ws . store plugin_re

let plugins_multiline = sep . counter "items" . [ seq "items" . new_ln_sep]*

let sto_multiline = optspace . Sep.opt_space . store (Rx.space_in . (/[ \t]*\n/ . Rx.space . Rx.space_in)*)

(************************************************************************
 * Group:                       ENTRY
 *************************************************************************)

let set_option = Util.del_str "set "
let no_inline_comment_entry (kw:regexp) (sep:lens) (comment:lens) =
                         [ set_option . key debug_kw . sep . IniFile.sto_to_eol . eol ]
                         | [ key plugins_kw . plugins_multiline . eol]
                         | [ key kw . sep . IniFile.sto_to_eol? . eol ]
                         | comment

let entry   = no_inline_comment_entry normal_opts sep comment

(************************************************************************
 *                        RECORD
 *************************************************************************)

let title   = IniFile.title IniFile.record_re

let record  = IniFile.record title entry

(************************************************************************
 * Group:                        LENS & FILTER
 *************************************************************************)

let lns = IniFile.lns record comment

let filter = incl "/etc/baruwa/*.ini"

let xfm = transform lns filter

