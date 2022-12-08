(*
Module: Postfix_Passwordmap
  Parses /etc/postfix/*passwd

Author: Anton Baranov <abaranov@linuxfoundation.org>

About: Reference
  This lens tries to keep as close as possible to `man 5 postconf` and
  http://www.postfix.org/SASL_README.html#client_sasl_enable where possible.

About: License
   This file is licenced under the LGPL v2+, like the rest of Augeas.

About: Configuration files
   This lens applies to /etc/postfix/*passwd. See <filter>.

About: Examples
   The <Test_Postfix_Passwordmap> file contains various examples and tests.
*)

module Postfix_Passwordmap =

autoload xfm

(* View: space_or_eol *)
let space_or_eol = del /([ \t]*\n)?[ \t]+/ " "

(* View: word *)
let word = store /[A-Za-z0-9@_\+\*.-]+/

(* View: colon *)
let colon = Sep.colon

(* View: username *)
let username = [ label "username" . word ]

(* View: password *)
let password = [ label "password" . (store Rx.space_in)? ]

(* View: record *)
let record = [ label "pattern" . store /\[?[A-Za-z0-9@\*.-]+\]?(:?[A-Za-z0-9]*)*/
             . space_or_eol . username . colon . password
             . Util.eol ]

(* View: lns *)
let lns = (Util.empty | Util.comment | record)*

(* Variable: filter *)
let filter = incl "/etc/postfix/*passwd"
           . incl "/usr/local/etc/postfix/*passwd"

let xfm = transform lns filter
