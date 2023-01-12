(*
Module: Rancid
  Parses RANCiD router database

Author: Matt Dainty <matt@bodgit-n-scarper.com>

About: Reference
       - man 5 router.db

Each line represents a record consisting of a number of ';'-separated fields
the first of which is the IP/Hostname of the device, followed by the type, its
state and optionally a comment.

*)

module Rancid =
  autoload xfm

  let sep     = Util.del_str ";"
  let field   = /[^;#\n]+/
  let comment = [ label "comment" . store /[^;#\n]*/ ]
  let eol     = Util.del_str "\n"
  let record  = [ label "device" . store field . sep . [ label "type" . store field ] . sep . [ label "state" . store field ] . ( sep . comment )? . eol ]

  let lns = ( Util.empty | Util.comment_generic /#[ \t]*/ "# " | record )*

  let filter = incl "/var/rancid/*/router.db"
             . Util.stdexcl

  let xfm = transform lns filter

(* Local Variables: *)
(* mode: caml       *)
(* End:             *)
