(*
Module: Hostname
  Parses /etc/hostname and /etc/mailname

Author: Raphael Pinson <raphink@gmail.com>

About: License
   This file is licenced under the LGPL v2+, like the rest of Augeas.
*)


module Hostname =
autoload xfm

(* View: lns *)
let lns = [ label "hostname" . store Rx.word . Util.eol ] | Util.empty

(* View: filter *)
let filter = incl "/etc/hostname"
           . incl "/etc/mailname"

let xfm = transform lns filter
