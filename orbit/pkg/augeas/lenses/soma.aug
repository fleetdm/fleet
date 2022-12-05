(* Soma module for Augeas
 Author: Free Ekanayaka <free@64studio.com>

 Reference: man 5 soma.cfg

*)

module Soma =

   autoload xfm

(************************************************************************
 *                           USEFUL PRIMITIVES
 *************************************************************************)

let eol        = Util.eol
let comment    = Util.comment
let empty      = Util.empty

let sep_eq     = del /[ \t]*=[ \t]*/ " = "

let sto_to_eol = store /([^ \t\n].*[^ \t\n]|[^ \t\n])/

let word       = /[A-Za-z0-9_.-]+/

(************************************************************************
 *                               ENTRIES
 *************************************************************************)

let entry     = [ key word
                . sep_eq
                . sto_to_eol
                . eol ]

(************************************************************************
 *                                LENS
 *************************************************************************)

let lns        = (comment|empty|entry) *

let filter     = incl "/etc/somad/soma.cfg"

let xfm        = transform lns filter
