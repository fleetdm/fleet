(*
Module: PamConf
  Parses /etc/pam.conf files

Author: Dominic Cleal <dcleal@redhat.com>

About: Reference
  This lens tries to keep as close as possible to `man pam.conf` where
  possible.

About: Licence
  This file is licensed under the LGPL v2+, like the rest of Augeas.

About: Lens Usage

About: Configuration files
  This lens applies to /etc/pam.conf. See <filter>.
*)
module PamConf =
  autoload xfm

(************************************************************************
 * Group:                 USEFUL PRIMITIVES
 *************************************************************************)

let indent  = Util.indent

let comment = Util.comment

let empty   = Util.empty

let include = Pam.include

let service = Rx.word

(************************************************************************
 * Group:                 LENSES
 *************************************************************************)

let record  = [ seq "record" . indent .
              [ label "service" . store service ] .
              Sep.space .
              Pam.record ]

let lns = ( empty | comment | include | record ) *

let filter = incl "/etc/pam.conf"

let xfm = transform lns filter

(* Local Variables: *)
(* mode: caml       *)
(* End:             *)
