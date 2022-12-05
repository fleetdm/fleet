(*
Module: AptConf
 Parses /etc/apt/apt.conf and /etc/apt/apt.conf.d/*

Author: Raphael Pinson <raphink@gmail.com>

About: Reference
 This lens tries to keep as close as possible to `man 5 apt.conf`
where possible.

About: License
  This file is licenced under the LGPL v2+, like the rest of Augeas.

About: Lens Usage
  To be documented

About: Configuration files
  This lens applies to /etc/apt/apt.conf and /etc/apt/apt.conf.d/*.
See <filter>.
*)


module AptConf =
 autoload xfm

(************************************************************************
 * Group:                 USEFUL PRIMITIVES
 *************************************************************************)

(* View: eol
   And <Util.eol> end of line *)
let eol = Util.eol

(* View: empty
   A C-style empty line *)
let empty = Util.empty_any

(* View: indent
   An indentation *)
let indent = Util.indent

(* View: comment_simple
   A one-line comment, C-style *)
let comment_simple = Util.comment_c_style_or_hash

(* View: comment_multi
   A multiline comment, C-style *)
let comment_multi = Util.comment_multiline

(* View: comment
   A comment, either <comment_simple> or <comment_multi> *)
let comment = comment_simple | comment_multi


(************************************************************************
 * Group:                 ENTRIES
 *************************************************************************)

(* View: name_re
   Regex for entry names *)
let name_re = /[A-Za-z][A-Za-z-]*/

(* View: name_re_colons
   Regex for entry names with colons *)
let name_re_colons = /[A-Za-z][A-Za-z:-]*/


(* View: entry
   An apt.conf entry, recursive

   WARNING:
     This lens exploits a put ambiguity
     since apt.conf allows for both
     APT { Clean-Installed { "true" } }
     and APT::Clean-Installed "true";
     but we're choosing to map them the same way

     The recursive lens doesn't seem
     to care and defaults to the first
     item in the union.

     This is why the APT { Clean-Installed { "true"; } }
     form is listed first, since it supports
     all subnodes (which Dpkg::Conf) doesn't.

     Exchanging these two expressions in the union
     makes tests fails since the tree cannot
     be mapped back.

     This situation results in existing
     configuration being modified when the
     associated tree is modified. For example,
     changing the value of
     APT::Clean-Installed "true"; to "false"
     results in
     APT { Clean-Installed "false"; }
     (see unit tests)
 *)
let rec entry_noeol =
 let value =
    Util.del_str "\"" . store /[^"\n]+/
                      . del /";?/ "\";" in
 let opt_eol = del /[ \t\n]*/ "\n" in
 let long_eol = del /[ \t]*\n+/ "\n" in
 let list_elem = [ opt_eol . label "@elem" . value ] in
 let eol_comment = del /([ \t\n]*\n)?/ "" . comment in
     [ key name_re . Sep.space . value ]
   | [ key name_re . del /[ \t\n]*\{/ " {" .
         ( (opt_eol . entry_noeol) |
           list_elem |
           eol_comment
           )* .
         del /[ \t\n]*\};?/ "\n};" ]
   | [ key name_re . Util.del_str "::" . entry_noeol ]

let entry = indent . entry_noeol . eol


(* View: include
   A file inclusion
   /!\ The manpage is not clear on the syntax *)
let include =
 [ indent . key "#include" . Sep.space
          . store Rx.fspath . eol ]


(* View: clear
   A list of variables to clear
   /!\ The manpage is not clear on the syntax *)
let clear =
 let name = [ label "name" . store name_re_colons ] in
 [ indent . key "#clear" . Sep.space
          . Build.opt_list name Sep.space
          . eol ]


(************************************************************************
 * Group:                 LENS AND FILTER
 *************************************************************************)

(* View: lns
    The apt.conf lens *)
let lns = (empty|comment|entry|include|clear)*


(* View: filter *)
let filter = incl "/etc/apt/apt.conf"
   . incl "/etc/apt/apt.conf.d/*"
   . Util.stdexcl

let xfm = transform lns filter
