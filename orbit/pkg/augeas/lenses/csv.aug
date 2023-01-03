(*
Module: CSV
  Generic CSV lens collection

Author: Raphael Pinson <raphael.pinson@camptocamp.com>

About: Reference
  https://tools.ietf.org/html/rfc4180

About: License
   This file is licenced under the LGPL v2+, like the rest of Augeas.

About: Lens Usage
   To be documented

About: Configuration files

About: Examples
   The <Test_CSV> file contains various examples and tests.

Caveats:
   No support for files without an ending CRLF
*)
module CSV =

(* View: eol *)
let eol = Util.del_str "\n"

(* View: comment *)
let comment = Util.comment
            | [ del /#[ \t]*\r?\n/ "#\n" ]

(* View: entry
     An entry of fields, quoted or not *)
let entry (sep_str:string) =
  let field = [ seq "field" . store (/[^"#\r\n]/ - sep_str)* ]
            | [ seq "field" . store /("[^"#]*")+/ ]
  in let sep = Util.del_str sep_str
  in [ seq "entry" . counter "field" . Build.opt_list field sep . eol ]

(* View: lns
     The generic lens, taking the separator as a parameter *)
let lns_generic (sep:string) = (comment | entry sep)*

(* View: lns
     The comma-separated value lens *)
let lns = lns_generic ","

(* View: lns_semicol
     A semicolon-separated value lens *)
let lns_semicol = lns_generic ";"
