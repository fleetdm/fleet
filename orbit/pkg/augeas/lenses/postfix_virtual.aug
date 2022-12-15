(*
Module: Postfix_Virtual
  Parses /etc/postfix/virtual

Author: Raphael Pinson <raphael.pinson@camptocamp.com>

About: Reference
  This lens tries to keep as close as possible to `man 5 virtual` where possible.

About: License
   This file is licenced under the LGPL v2+, like the rest of Augeas.

About: Lens Usage
   To be documented

About: Configuration files
   This lens applies to /etc/postfix/virtual. See <filter>.

About: Examples
   The <Test_Postfix_Virtual> file contains various examples and tests.
*)

module Postfix_Virtual =

autoload xfm

(* Variable: space_or_eol_re *)
let space_or_eol_re = /([ \t]*\n)?[ \t]+/

(* View: space_or_eol *)
let space_or_eol (sep:regexp) (default:string) =
  del (space_or_eol_re? . sep . space_or_eol_re?) default

(* View: word *)
let word = store /[A-Za-z0-9@\*.+=_-]+/

(* View: comma *)
let comma = space_or_eol "," ", "

(* View: destination *)
let destination = [ label "destination" . word ]

(* View: record *)
let record =
  let destinations = Build.opt_list destination comma
  in [ label "pattern" . word
     . space_or_eol Rx.space " " . destinations
     . Util.eol ]

(* View: lns *)
let lns = (Util.empty | Util.comment | record)*

(* Variable: filter *)
let filter = incl "/etc/postfix/virtual"
           . incl "/usr/local/etc/postfix/virtual"

let xfm = transform lns filter
