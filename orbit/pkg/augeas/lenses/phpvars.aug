(* Phpvars module for Augeas
 Author: Free Ekanayaka <free@64studio.com>

 Reference: PHP syntax

*)

module Phpvars =

   autoload xfm

(************************************************************************
 *                           USEFUL PRIMITIVES
 *************************************************************************)

let eol        = Util.eol
let empty      = Util.empty_c_style

let open_php   = del /<\?(php)?[ \t]*\n/i "<?php\n"
let close_php  = del /([ \t]*(php)?\?>\n[ \t\n]*)?/i "php?>\n"
let sep_eq     = del /[ \t\n]*=[ \t\n]*/ " = "
let sep_opt_spc = Sep.opt_space
let sep_spc    = Sep.space
let sep_dollar = del /\$/ "$"
let sep_scl    = del /[ \t]*;/ ";"

let chr_blank = /[ \t]/
let chr_nblank = /[^ \t\n]/
let chr_any    = /./
let chr_star   = /\*/
let chr_nstar  = /[^* \t\n]/
let chr_slash  = /\//
let chr_nslash = /[^\/ \t\n]/
let chr_variable = /\$[A-Za-z0-9'"_:-]+/

let sto_to_scl = store (/([^ \t\n].*[^ \t\n;]|[^ \t\n;])/ - /.*;[ \t]*(\/\/|#).*/) (* " *)
let sto_to_eol = store /([^ \t\n].*[^ \t\n]|[^ \t\n])/

(************************************************************************
 *                              COMMENTS
 *************************************************************************)

(* Both c-style and shell-style comments are valid
   Default to c-style *)
let comment_one_line = Util.comment_generic /[ \t]*(\/\/|#)[ \t]*/ "// "

let comment_eol = Util.comment_generic /[ \t]*(\/\/|#)[ \t]*/ " // "

let comment      = Util.comment_multiline | comment_one_line

let eol_or_comment = eol | comment_eol


(************************************************************************
 *                               ENTRIES
 *************************************************************************)

let simple_line (kw:regexp) (lns:lens) = [ key kw
                 . lns
                 . sep_scl
                 . eol_or_comment ]

let global     = simple_line "global" (sep_opt_spc . sep_dollar . sto_to_scl)

let assignment =
  let arraykey = [ label "@arraykey" . store /\[[][A-Za-z0-9'"_:-]+\]/ ] in (* " *)
  simple_line chr_variable (arraykey? . (sep_eq . sto_to_scl))

let variable = Util.indent . assignment

let classvariable =
  Util.indent . del /(public|var)/ "public" . Util.del_ws_spc . assignment

let include = simple_line "@include" (sep_opt_spc . sto_to_scl)

let generic_function (kw:regexp) (lns:lens) =
  let lbracket = del /[ \t]*\([ \t]*/ "(" in
  let rbracket = del /[ \t]*\)/ ")" in
    simple_line kw (lbracket . lns . rbracket)

let define     =
  let variable_re = /[A-Za-z0-9'_:-]+/ in
  let quote = del /["']/ "'" in
  let sep_comma = del /["'][ \t]*,[ \t]*/ "', " in
  let sto_to_rbracket = store (/[^ \t\n][^\n]*[^ \t\n\)]|[^ \t\n\)]/
                             - /.*;[ \t]*(\/\/|#).*/) in
    generic_function "define" (quote . store variable_re . sep_comma
                                     . [ label "value" . sto_to_rbracket ])

let simple_function (kw:regexp) =
  let sto_to_rbracket = store (/[^ \t\n][^\n]*[^ \t\n\)]|[^ \t\n\)]/
                             - /.*;[ \t]*(\/\/|#).*/) in
    generic_function kw sto_to_rbracket

let entry      = Util.indent
               . ( global
                 | include
                 | define
                 | simple_function "include"
                 | simple_function "include_once"
                 | simple_function "echo" )


let class =
  let classname = key /[A-Za-z0-9'"_:-]+/ in (* " *)
  del /class[ \t]+/ "class " .
  [ classname . Util.del_ws_spc . del "{" "{" .
    (empty|comment|entry|classvariable)*
  ] . del "}" "}"

(************************************************************************
 *                                LENS
 *************************************************************************)

let lns        = open_php . (empty|comment|entry|class|variable)* . close_php

let filter     = incl "/etc/squirrelmail/config.php"

let xfm        = transform lns filter
