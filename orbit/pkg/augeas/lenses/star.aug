(*
Module: Star
  Parses star's configuration file

Author: Cedric Bosdonnat <cbosdonnat@suse.com>

About: Reference
    This lens is based on star(1)

About: License
  This file is licensed under the LGPL v2+, like the rest of Augeas.
*)

module Star =
  autoload xfm

let sto_to_tab = store Rx.no_spaces

let size  = Build.key_value_line "STAR_FIFOSIZE" Sep.space_equal ( store /[0-9x*.a-z]+/ )
let size_max   = Build.key_value_line "STAR_FIFOSIZE_MAX" Sep.space_equal ( store /[0-9x*.a-z]+/ )
let archive = Build.key_value_line ( "archive". /[0-7]/ ) Sep.equal
               ( [ label "device" . sto_to_tab ] . Sep.tab .
                 [ label "block" . sto_to_tab ] . Sep.tab .
                 [ label "size" . sto_to_tab ] . ( Sep.tab .
                 [ label "istape" . sto_to_tab ] )? )

let lns = ( size | size_max | archive | Util.comment | Util.empty )*

let filter = incl "/etc/default/star"

let xfm = transform lns filter
