(*
Module: Inputrc
  Parses /etc/inputrc

Author: Raphael Pinson <raphink@gmail.com>

About: Reference
  This lens tries to keep as close as possible to `man 3 readline` where possible.

About: License
   This file is licenced under the LGPL v2+, like the rest of Augeas.

About: Lens Usage
   To be documented

About: Configuration files
   This lens applies to /etc/inputrc. See <filter>.

About: Examples
   The <Test_Inputrc> file contains various examples and tests.
*)

module Inputrc =

autoload xfm

(* View: entry
     An inputrc mapping entry *)
let entry =
   let mapping = [ label "mapping" . store /[A-Za-z0-9_."\*\/+\,\\-]+/ ]
   in [ label "entry"
      . Util.del_str "\"" . store /[^" \t\n]+/
      . Util.del_str "\":" . Sep.space
      . mapping
      . Util.eol ]

(* View: variable
     An inputrc variable declaration *)
let variable = [ Util.del_str "set" . Sep.space
               . key (Rx.word - "entry") . Sep.space
               . store Rx.word . Util.eol ]

(* View: condition
     An "if" declaration, recursive *)
let rec condition = [ Util.del_str "$if" . label "@if"
                    . Sep.space . store Rx.space_in . Util.eol
                    . (Util.empty | Util.comment | condition | variable | entry)*
                    . [ Util.del_str "$else" . label "@else" . Util.eol
                      . (Util.empty | Util.comment | condition | variable | entry)* ] ?
                    . Util.del_str "$endif" . Util.eol ]

(* View: lns
     The inputrc lens *)
let lns = (Util.empty | Util.comment | condition | variable | entry)*

(* Variable: filter *)
let filter = incl "/etc/inputrc"

let xfm = transform lns filter
