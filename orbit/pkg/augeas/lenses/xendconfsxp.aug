module Xendconfsxp =
    autoload xfm

let spc1 = /[ \t\n]+/
let ws = del spc1 " "

let lbrack = del /[ \t]*\([ \t\n]*/ "("
let rbrack = del /[ \t\n]*\)/ ")"

let empty_line = [ del /[ \t]*\n/ "\n" ]

let no_ws_comment =
    [ label "#comment" . del /#[ \t]*/ "# " . store /[^ \t]+[^\n]*/ . del /\n/ "\n" ]

let standalone_comment = [ label "#scomment" . del /#/ "#" . store /.*/ . del /\n/ "\n" ]
(* Minor bug: The initial whitespace is stored, not deleted. *)

let ws_and_comment = ws . no_ws_comment

(* Either a word or a quoted string *)
let str_store = store /[A-Za-z0-9_.\/-]+|\"([^\"\\\\]|(\\\\.))*\"|'([^'\\\\]|(\\\\.))*'/

let str = [ label "string" . str_store ]

let var_name = key Rx.word

let rec thing =
    let array = [ label "array" . lbrack . Build.opt_list thing ws . ws_and_comment? . rbrack ] in
    let str = [ label "item" . str_store ] in
    str | array

let sexpr = [ lbrack . var_name . ws . no_ws_comment? . thing . ws_and_comment? . rbrack ]

let lns = ( empty_line | standalone_comment | sexpr ) *

let filter = incl "xend-config.sxp"
let xfm = transform lns filter
