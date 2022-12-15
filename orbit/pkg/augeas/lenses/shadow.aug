(*
 Module: Shadow
 Parses /etc/shadow

 Author: Lorenzo M. Catucci <catucci@ccd.uniroma2.it>

 Original Author: Free Ekanayaka <free@64studio.com>

 About: Reference

   - man 5 shadow
   - man 3 getspnam

 About: License
   This file is licensed under the LGPL v2+, like the rest of Augeas.

 About:

 Each line in the shadow files represents the additional shadow-defined attributes
 for the corresponding user, as defined in the passwd file.

*)

module Shadow =

   autoload xfm

(************************************************************************
 *                           USEFUL PRIMITIVES
 *************************************************************************)

let eol        = Util.eol
let comment    = Util.comment
let empty      = Util.empty
let dels       = Util.del_str

let colon      = Sep.colon

let word       = Rx.word
let integer    = Rx.integer

let sto_to_col = Passwd.sto_to_col
let sto_to_eol = Passwd.sto_to_eol

(************************************************************************
 * Group:                        ENTRIES
 *************************************************************************)
(* Common for entry and nisdefault *)
let common =  [ label "lastchange_date" . store integer? . colon ]
            . [ label "minage_days"     . store integer? . colon ]
            . [ label "maxage_days"     . store integer? . colon ]
            . [ label "warn_days"       . store integer? . colon ]
            . [ label "inactive_days"   . store integer? . colon ]
            . [ label "expire_date"     . store integer? . colon ]
            . [ label "flag"            . store integer? ]
              
(* View: entry *)
let entry  = [ key word
               . colon
               . [ label "password" . sto_to_col? . colon ]
               . common
               . eol ]

let nisdefault =
           let overrides =
             colon
               . [ label "password" . store word? . colon ]
               . common in
           [ dels "+" . label "@nisdefault" . overrides? . eol ]

(************************************************************************
 *                                LENS
 *************************************************************************)

let lns        = (comment|empty|entry|nisdefault) *

let filter
               = incl "/etc/shadow"
               . Util.stdexcl

let xfm        = transform lns filter
