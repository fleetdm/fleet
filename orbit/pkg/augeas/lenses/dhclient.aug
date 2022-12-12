(* Intefraces module for Augeas
   Author: Free Ekanayaka <free@64studio.com>

   Reference: man dhclient.conf
   The only difference with the reference syntax is that this lens assumes
   that statements end with a new line, while the reference syntax allows
   new statements to be started right after the trailing ";" of the
   previous statement. This should not be a problem in real-life
   configuration files as statements get usually split across several
   lines, rather than merged in a single one.

*)

module Dhclient =

   autoload xfm

(************************************************************************
 *                           USEFUL PRIMITIVES
 *************************************************************************)

let eol               = Util.eol
let comment           = Util.comment
let comment_or_eol    = Util.comment_or_eol
let empty             = Util.empty

(* Define separators *)
let sep_spc           = del /[ \t\n]+/ " "
let sep_scl           = del /[ \t]*;/ ";"
let sep_obr           = del /[ \t\n]*\{\n]*/ " {\n"
let sep_cbr           = del /[ \t\n]*\}/ " }"
let sep_com           = del /[ \t\n]*,[ \t\n]*/ ","
let sep_slh           = del "\/" "/"
let sep_col           = del ":" ":"
let sep_eq            = del /[ \t]*=[ \t]*/ "="

(* Define basic types *)
let word              = /[A-Za-z0-9_.-]+(\[[0-9]+\])?/

(* Define fields *)

(* TODO: there could be a " " in the middle of a value ... *)
let sto_to_spc        = store /[^\\#,;{}" \t\n]+|"[^\\#"\n]+"/
let sto_to_spc_noeval = store /[^=\\#,;{}" \t\n]|[^=\\#,;{}" \t\n][^\\#,;{}" \t\n]*|"[^\\#"\n]+"/
let sto_to_scl        = store /[^ \t\n][^;\n]+[^ \t]|[^ \t;\n]+/
let rfc_code          = [ key "code" . sep_spc . store word ]
                      . sep_eq
                      . [ label "value" . sto_to_scl ]
let eval              = [ label "#eval" . Sep.equal . sep_spc . sto_to_scl ]
let sto_number        = store /[0-9][0-9]*/

(************************************************************************
 *                         SIMPLE STATEMENTS
 *************************************************************************)

let stmt_simple_re    = "timeout"
                      | "retry"
                      | "select-timeout"
                      | "reboot"
                      | "backoff-cutoff"
                      | "initial-interval"
                      | "do-forward-updates"
                      | "reject"

let stmt_simple       = [ key stmt_simple_re
                        . sep_spc
                        . sto_to_spc
                        . sep_scl
                        . comment_or_eol ]


(************************************************************************
 *                          ARRAY STATEMENTS
 *************************************************************************)

(* TODO: the array could also be empty, like in the request statement *)
let stmt_array_re     = "media"
                      | "request"
                      | "require"

let stmt_array        = [ key stmt_array_re
                        . sep_spc
                        . counter "stmt_array"
                        . [ seq "stmt_array" . sto_to_spc ]
                        . [ sep_com . seq "stmt_array" . sto_to_spc ]*
                        . sep_scl . comment_or_eol ]

(************************************************************************
 *                          HASH STATEMENTS
 *************************************************************************)


let stmt_hash_re      = "send"
                      | "option"

let stmt_args         = ( [ key word . sep_spc . sto_to_spc_noeval ]
                          | [ key word . sep_spc . (rfc_code|eval) ] )
                        . sep_scl
                        . comment_or_eol

let stmt_hash         = [ key stmt_hash_re
                        . sep_spc
                        . stmt_args ]

let stmt_opt_mod_re   = "append"
                      | "prepend"
                      | "default"
                      | "supersede"

let stmt_opt_mod      = [ key stmt_opt_mod_re
                        . sep_spc
                        . stmt_args ]

(************************************************************************
 *                         BLOCK STATEMENTS
 *************************************************************************)

let stmt_block_re     = "interface"
                      | "lease"
                      | "alias"

let stmt_block_opt_re = "interface"
                      | "script"
                      | "bootp"
                      | "fixed-address"
                      | "filename"
                      | "server-name"
                      | "medium"
                      | "vendor option space"

(* TODO: some options could take no argument like bootp *)
let stmt_block_opt    = [ key stmt_block_opt_re
                         . sep_spc
                         . sto_to_spc
                         . sep_scl
                         . comment_or_eol ]

let stmt_block_date_re
                      = "renew"
                      | "rebind"
                      | "expire"

let stmt_block_date   = [ key stmt_block_date_re
                        . [ sep_spc . label "weekday" . sto_number ]
                        . [ sep_spc . label "year"    . sto_number ]
                        . [ sep_slh . label "month"   . sto_number ]
                        . [ sep_slh . label "day"     . sto_number ]
                        . [ sep_spc . label "hour"    . sto_number ]
                        . [ sep_col . label "minute"  . sto_number ]
                        . [ sep_col . label "second"  . sto_number ]
                        . sep_scl
                        . comment_or_eol ]

let stmt_block_arg    = sep_spc . sto_to_spc

let stmt_block_entry  = sep_spc
                      . ( stmt_array
                        | stmt_hash
                        | stmt_opt_mod
                        | stmt_block_opt
                        | stmt_block_date )

let stmt_block        = [ key stmt_block_re
                        . stmt_block_arg?
                        . sep_obr
                        . stmt_block_entry+
                        . sep_cbr
                        . comment_or_eol ]

(************************************************************************
 *                              LENS & FILTER
 *************************************************************************)

let statement = (stmt_simple|stmt_opt_mod|stmt_array|stmt_hash|stmt_block)

let lns               = ( empty
                        | comment
                        | statement )*

let filter            = incl "/etc/dhcp3/dhclient.conf"
                      . incl "/etc/dhcp/dhclient.conf"
                      . incl "/etc/dhclient.conf"

let xfm                = transform lns filter
