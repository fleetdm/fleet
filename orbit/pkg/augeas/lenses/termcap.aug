(*
Module: Termcap
  Parses termcap capability database

Author: Matt Dainty <matt@bodgit-n-scarper.com>

About: Reference
       - man 5 termcap

Each line represents a record consisting of a number of ':'-separated fields
the first of which is the name or identifier for the record. The name can
optionally be split by '|' and each subsequent value is considered an alias
of the first. Records can be split across multiple lines with '\'.

*)

module Termcap =
  autoload xfm

  (* All termcap capabilities are two characters, optionally preceded by *)
  (* upto two periods and the only types are boolean, numeric or string  *)
  let cfield = /\.{0,2}([a-zA-Z0-9]{2}|[@#%&*!][a-zA-Z0-9]|k;)(#?@|#[0-9]+|=([^:\\\\^]|\\\\[0-7]{3}|\\\\[:bBcCeEfFnNrRstT0\\^]|\^.)*)?/

  let lns = ( Util.empty | Getcap.comment | Getcap.record cfield )*

  let filter = incl "/etc/termcap"
             . incl "/usr/share/misc/termcap"
             . Util.stdexcl

  let xfm = transform lns filter

(* Local Variables: *)
(* mode: caml       *)
(* End:             *)
