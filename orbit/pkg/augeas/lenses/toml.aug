(*
Module: Toml
  Parses TOML files

Author: Raphael Pinson <raphael.pinson@camptocamp.com>

About: Reference
  https://toml.io/en/v1.0.0

About: License
   This file is licensed under the LGPL v2+, like the rest of Augeas.

About: Lens Usage
   To be documented

About: Configuration files
   This lens applies to TOML files.

About: Examples
   The <Test_Toml> file contains various examples and tests.
*)

module Toml =

(* Group: base definitions *)

(* View: comment
     A simple comment *)
let comment  = IniFile.comment "#" "#"

(* View: empty
     An empty line *)
let empty = Util.empty_dos

(* View: eol
     An end of line *)
let eol = Util.doseol


(* Group: value entries *)

let bare_re_noquot = (/[^][", \t\r\n]/ - "#")
let bare_re = (/[^][,\r=]/ - "#")+
let no_quot = /[^]["\r\n]*/
let bare = Quote.do_dquote_opt_nil (store (bare_re_noquot . (bare_re* . bare_re_noquot)?))
let quoted = Quote.do_dquote (store (/[^"]/ . "#"* . /[^"]/))

let ws = del /[ \t\n]*/ ""

let space_or_empty = [ del /[ \t\n]+/ " " ]

let comma = Util.del_str "," . (space_or_empty | comment)?
let lbrace = Util.del_str "{" . (space_or_empty | comment)?
let rbrace = Util.del_str "}"
let lbrack = Util.del_str "[" . (space_or_empty | comment)?
let rbrack = Util.del_str "]"

(* This follows the definition of 'string' at https://www.json.org/
   It's a little wider than what's allowed there as it would accept
   nonsensical \u escapes *)
let triple_dquote = Util.del_str "\"\"\""
let str_store = Quote.dquote . store /([^\\"]|\\\\["\/bfnrtu\\])*/ . Quote.dquote

let str_store_multi = triple_dquote . eol
                  . store /([^\\"]|\\\\["\/bfnrtu\\])*/
                  . del /\n[ \t]*/ "\n" . triple_dquote

let str_store_literal = Quote.squote . store /([^\\']|\\\\['\/bfnrtu\\])*/ . Quote.squote

let integer =
     let base10 = /[+-]?[0-9_]+/
  in let hex = /0x[A-Za-z0-9]+/
  in let oct = /0o[0-7]+/
  in let bin = /0b[01]+/
  in [ label "integer" . store (base10 | hex | oct | bin) ]

let float =
     let n = /[0-9_]+/
  in let pm = /[+-]?/
  in let z = pm . n
  in let decim = "." . n
  in let exp = /[Ee]/ . z
  in let num = z . decim | z . exp | z . decim . exp
  in let inf = pm . "inf"
  in let nan = pm . "nan"
  in [ label "float" . store (num | inf | nan) ]

let str = [ label "string" . str_store ]

let str_multi = [ label "string_multi" . str_store_multi ]

let str_literal = [ label "string_literal" . str_store_literal ]

let bool (r:regexp) = [ label "bool" . store r ]


let date_re = /[0-9]{4}-[0-9]{2}-[0-9]{2}/
let time_re = /[0-9]{1,2}:[0-9]{2}:[0-9]{2}(\.[0-9]+)?[A-Z]*/

let datetime = [ label "datetime" . store (date_re . /[T ]/ . time_re) ]
let date = [ label "date" . store date_re ]
let time = [ label "time" . store time_re ]

let norec = str | str_multi | str_literal
          | integer | float | bool /true|false/
          | datetime | date |  time

let array (value:lens) = [ label "array" . lbrack
               . ( ( Build.opt_list value comma . space_or_empty? . rbrack )
                   | rbrack ) ]

let array_norec = array norec

(* This is actually no real recursive array, instead it is one or two dimensional
   For more info on this see https://github.com/hercules-team/augeas/issues/715 *)
let array_rec = array (norec | array_norec)

let entry_base (value:lens) = [ label "entry" . store Rx.word . Sep.space_equal . value ]

let inline_table (value:lens) = [ label "inline_table" . lbrace
                   . ( (Build.opt_list (entry_base value) comma . space_or_empty? . rbrace)
                      | rbrace ) ]

let entry = [ label "entry" . Util.indent . store Rx.word . Sep.space_equal
            . (norec | array_rec | inline_table (norec|array_norec)) . (eol | comment) ]

(* Group: tables *)

(* View: table_gen
     A generic table *)
let table_gen (name:string) (lbrack:string) (rbrack:string) =
     let title = Util.indent . label name
               . Util.del_str lbrack
               . store /[^]\r\n.]+(\.[^]\r\n.]+)*/
               . Util.del_str rbrack . eol
  in [ title . (entry|empty|comment)* ]

(* View: table
     A table or array of tables *)
let table = table_gen "table" "[" "]"
          | table_gen "@table" "[[" "]]"

(* Group: lens *)

(* View: lns
     The Toml lens *)
let lns = (entry | empty | comment)* . table*
