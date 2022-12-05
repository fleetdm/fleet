(* Webmin module for Augeas
 Author: Free Ekanayaka <free@64studio.com>

 Reference:

*)

module Webmin =

   autoload xfm

(************************************************************************
 *                           USEFUL PRIMITIVES
 *************************************************************************)

let eol        = Util.eol
let comment    = Util.comment
let empty      = Util.empty

let sep_eq     = del /=/ "="

let sto_to_eol = store /([^ \t\n].*[^ \t\n]|[^ \t\n])/

let word       = /[A-Za-z0-9_.-]+/

(************************************************************************
 *                               ENTRIES
 *************************************************************************)

let entry     = [ key word
                . sep_eq
                . sto_to_eol?
                . eol ]

(************************************************************************
 *                                LENS
 *************************************************************************)

let lns        = (comment|empty|entry) *

let wm_incl (n:string)
               = (incl ("/etc/webmin/" . n))
let filter     = wm_incl "miniserv.conf"
               . wm_incl "ldap-useradmin/config"

let xfm        = transform lns filter
