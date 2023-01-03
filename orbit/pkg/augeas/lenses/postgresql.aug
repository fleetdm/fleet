(*
Module: Postgresql
  Parses postgresql.conf

Author: Raphael Pinson <raphink@gmail.com>

About: Reference
  http://www.postgresql.org/docs/current/static/config-setting.html

About: License
   This file is licenced under the LGPL v2+, like the rest of Augeas.

About: Lens Usage
   To be documented

About: Configuration files
   This lens applies to postgresql.conf. See <filter>.

About: Examples
   The <Test_Postgresql> file contains various examples and tests.
*)


module Postgresql =
  autoload xfm

(* View: sep
     Key and values are separated
     by either spaces or an equal sign *)
let sep = del /([ \t]+)|([ \t]*=[ \t]*)/ " = "

(* Variable: word_opt_quot_re
     Strings that don't require quotes *)
let word_opt_quot_re = /[A-Za-z][A-Za-z0-9_-]*/

(* View: word_opt_quot
     Storing a <word_opt_quot_re>, with or without quotes *)
let word_opt_quot = Quote.do_squote_opt (store word_opt_quot_re)

(* Variable: number_re
     A relative decimal number, optionally with unit *)
let number_re = Rx.reldecimal . /[kMG]?B|[m]?s|min|h|d/?

(* View: number
     Storing <number_re>, with or without quotes *)
let number = Quote.do_squote_opt (store number_re)

(* View: word_quot
     Anything other than <word_opt_quot> or <number>
     Quotes are mandatory *)
let word_quot =
     let esc_squot = /\\\\'/
  in let no_quot = /[^#'\n]/
  in let forbidden = word_opt_quot_re | number_re
  in let value = (no_quot|esc_squot)* - forbidden
  in Quote.do_squote (store value)

(* View: entry_gen
     Builder to construct entries *)
let entry_gen (lns:lens) =
  Util.indent . Build.key_value_line_comment Rx.word sep lns Util.comment_eol

(* View: entry *)
let entry = entry_gen number
          | entry_gen word_opt_quot
          | entry_gen word_quot    (* anything else *)

(* View: lns *)
let lns = (Util.empty | Util.comment | entry)*

(* Variable: filter *)
let filter = (incl "/var/lib/pgsql/data/postgresql.conf" .
              incl "/var/lib/pgsql/*/data/postgresql.conf" .
              incl "/var/lib/postgresql/*/data/postgresql.conf" .
              incl "/etc/postgresql/*/*/postgresql.conf" )

let xfm = transform lns filter

