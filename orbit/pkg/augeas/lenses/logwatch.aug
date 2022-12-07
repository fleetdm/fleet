(* Logwatch module for Augeas
 Author: Francois Lebel <francois@flebel.com>
 Based on the dnsmasq lens written by Free Ekanayaka.

 Reference: man logwatch (8)

 "Format is one option per line, legal options are the same
  as the long options legal on the command line. See
 "logwatch.pl --help" or "man 8 logwatch" for details."

*)

module Logwatch =

   autoload xfm

(************************************************************************
 *                           USEFUL PRIMITIVES
 *************************************************************************)

let eol        = Util.eol
let spc        = Util.del_ws_spc
let comment    = Util.comment
let empty      = Util.empty

let sep_eq     = del / = / " = "
let sto_to_eol = store /([^ \t\n].*[^ \t\n]|[^ \t\n])/

(************************************************************************
 *                               ENTRIES
 *************************************************************************)

let entry_re   = /[A-Za-z0-9._-]+/
let entry      = [ key entry_re . sep_eq . sto_to_eol . eol ]

(************************************************************************
 *                                LENS
 *************************************************************************)

let lns = (comment|empty|entry) *

let filter            = incl "/etc/logwatch/conf/logwatch.conf"
                      . excl ".*"
                      . Util.stdexcl

let xfm                = transform lns filter
