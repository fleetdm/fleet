(*
Module: Util
  Generic module providing useful primitives

Author: David Lutterkort

About: License
  This file is licensed under the LGPLv2+, like the rest of Augeas.
*)


module Util =


(*
Variable: del_str
  Delete a string and default to it

  Parameters:
     s:string - the string to delete and default to
*)
  let del_str (s:string) = del s s

(*
Variable: del_ws
  Delete mandatory whitespace
*)
  let del_ws = del /[ \t]+/

(*
Variable: del_ws_spc
  Delete mandatory whitespace, default to single space
*)
  let del_ws_spc = del_ws " "

(*
Variable: del_ws_tab
  Delete mandatory whitespace, default to single tab
*)
  let del_ws_tab = del_ws "\t"


(*
Variable: del_opt_ws
  Delete optional whitespace
*)
  let del_opt_ws = del /[ \t]*/


(*
Variable: eol
  Delete end of line, including optional trailing whitespace
*)
  let eol = del /[ \t]*\n/ "\n"

(*
Variable: doseol
  Delete end of line with optional carriage return,
  including optional trailing whitespace
*)
  let doseol = del /[ \t]*\r?\n/ "\n"
   

(*
Variable: indent
  Delete indentation, including leading whitespace
*)
  let indent = del /[ \t]*/ ""

(* Group: Comment
     This is a general definition of comment
     It allows indentation for comments, removes the leading and trailing spaces
     of comments and stores them in nodes, except for empty comments which are
     ignored together with empty lines
*)


(* View: comment_generic_seteol
  Map comments and set default comment sign
*)

  let comment_generic_seteol (r:regexp) (d:string) (eol:lens) =
    [ label "#comment" . del r d
        . store /([^ \t\r\n].*[^ \t\r\n]|[^ \t\r\n])/ . eol ]

(* View: comment_generic
  Map comments and set default comment sign
*)

  let comment_generic (r:regexp) (d:string) =
    comment_generic_seteol r d doseol

(* View: comment
  Map comments into "#comment" nodes
*)
  let comment = comment_generic /[ \t]*#[ \t]*/ "# "

(* View: comment_noindent
  Map comments into "#comment" nodes, without indentation
*)
  let comment_noindent = comment_generic /#[ \t]*/ "# "

(* View: comment_eol
  Map eol comments into "#comment" nodes
  Add a space before # for end of line comments
*)
  let comment_eol = comment_generic /[ \t]*#[ \t]*/ " # "

(* View: comment_or_eol
    A <comment_eol> or <eol>, with an optional empty comment *)
 let comment_or_eol = comment_eol | (del /[ \t]*(#[ \t]*)?\n/ "\n")

(* View: comment_multiline
    A C-style multiline comment *)
  let comment_multiline =
     let mline_re = (/[^ \t\r\n].*[^ \t\r\n]|[^ \t\r\n]/ - /.*\*\/.*/) in
     let mline = [ seq "mline"
                 . del /[ \t\r\n]*/ "\n"
                 . store mline_re ] in
     [ label "#mcomment" . del /[ \t]*\/\*/ "/*"
       . counter "mline"
       . mline . (eol . mline)*
       . del /[ \t\r\n]*\*\/[ \t]*\r?\n/ "\n*/\n" ]

(* View: comment_c_style
    A comment line, C-style *)
  let comment_c_style =
    comment_generic /[ \t]*\/\/[ \t]*/ "// "

(* View: comment_c_style_or_hash
    A comment line, C-style or hash *)
  let comment_c_style_or_hash =
    comment_generic /[ \t]*((\/\/)|#)[ \t]*/ "// "

(* View: empty_generic
  A generic definition of <empty>
  Map empty lines, including empty comments *)
  let empty_generic (r:regexp) =
    [ del r "" . del_str "\n" ]

(* Variable: empty_generic_re *)
  let empty_generic_re = /[ \t]*#?[ \t]*/

(* View: empty
  Map empty lines, including empty comments *)
  let empty = empty_generic empty_generic_re

(* Variable: empty_c_style_re *)
  let empty_c_style_re = /[ \t]*((\/\/)|(\/\*[ \t]*\*\/))?[ \t]*/

(* View: empty_c_style
  Map empty lines, including C-style empty comment *)
  let empty_c_style = empty_generic empty_c_style_re

(* View: empty_any
  Either <empty> or <empty_c_style> *)
  let empty_any = empty_generic (empty_generic_re | empty_c_style_re)

(* View: empty_generic_dos
  A generic definition of <empty> with dos newlines
  Map empty lines, including empty comments *)
  let empty_generic_dos (r:regexp) =
    [ del r "" . del /\r?\n/ "\n" ]

(* View: empty_dos *)
  let empty_dos =
    empty_generic_dos /[ \t]*#?[ \t]*/


(* View: Split *)
(* Split (SEP . ELT)* into an array-like tree where each match for ELT *)
(* appears in a separate subtree. The labels for the subtrees are      *)
(* consecutive numbers, starting at 0                                  *)
  let split (elt:lens) (sep:lens) =
    let sym = gensym "split" in
    counter sym . ( [ seq sym . sep . elt ] ) *

(* View: delim *)
  let delim (op:string) = del (/[ \t]*/ . op . /[ \t]*/)
                              (" " . op . " ")

(* Group: Exclusions

Variable: stdexcl
  Exclusion for files that are commonly not wanted/needed
*)
  let stdexcl = (excl "*~") .
    (excl "*.rpmnew") .
    (excl "*.rpmsave") .
    (excl "*.dpkg-old") .
    (excl "*.dpkg-new") .
    (excl "*.dpkg-bak") .
    (excl "*.dpkg-dist") .
    (excl "*.augsave") .
    (excl "*.augnew") .
    (excl "*.bak") .
    (excl "*.old") .
    (excl "#*#")
