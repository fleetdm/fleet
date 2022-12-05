(*
Module: Getcap
  Parses generic termcap-style capability databases

Author: Matt Dainty <matt@bodgit-n-scarper.com>

About: Reference
       - man 3 getcap
       - man 5 login.conf
       - man 5 printcap

Each line represents a record consisting of a number of ':'-separated fields
the first of which is the name or identifier for the record. The name can
optionally be split by '|' and each subsequent value is considered an alias
of the first. Records can be split across multiple lines with '\'.

See also the Rtadvd and Termcap modules which contain slightly more specific
grammars.

*)

module Getcap =
  autoload xfm

  (* Comments cannot have any leading characters *)
  let comment                = Util.comment_generic /#[ \t]*/ "# "

  let nfield                 = /[^#:\\\\\t\n|][^:\\\\\t\n|]*/

  (* field must not contain ':' *)
  let cfield                 = /[a-zA-Z0-9-]+([%^$#\\]?@|[%^$#\\=]([^:\\\\^]|\\\\[0-7]{1,3}|\\\\[bBcCeEfFnNrRtT\\^]|\^.)*)?/

  let csep                   = del /:([ \t]*\\\\\n[ \t]*:)?/ ":\\\n\t:"
  let nsep                   = Util.del_str "|"
  let name                   = [ label "name" . store nfield ]
  let capability (re:regexp) = [ label "capability" . store re ]
  let record (re:regexp)     = [ label "record" . name . ( nsep . name )* . ( csep . capability re )* . Sep.colon . Util.eol ]

  let lns = ( Util.empty | comment | record cfield )*

  let filter = incl "/etc/login.conf"
             . incl "/etc/printcap"
             . Util.stdexcl

  let xfm = transform lns filter

(* Local Variables: *)
(* mode: caml       *)
(* End:             *)
