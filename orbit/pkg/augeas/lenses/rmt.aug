(*
Module: Rmt
  Parses rmt's configuration file

Author: Dominic Cleal <dcleal@redhat.com>

About: Reference
    This lens is based on rmt(1)

About: License
  This file is licensed under the LGPL v2+, like the rest of Augeas.
*)

module Rmt =
  autoload xfm

let sto_to_tab = store Rx.no_spaces

let debug  = Build.key_value_line "DEBUG" Sep.equal ( store Rx.fspath )
let user   = Build.key_value_line "USER" Sep.equal sto_to_tab
let access = Build.key_value_line "ACCESS" Sep.equal
               ( [ label "name" . sto_to_tab ] . Sep.tab .
                 [ label "host" . sto_to_tab ] . Sep.tab .
                 [ label "path" . sto_to_tab ] )

let lns = ( debug | user | access | Util.comment | Util.empty )*

let filter = incl "/etc/default/rmt"

let xfm = transform lns filter
