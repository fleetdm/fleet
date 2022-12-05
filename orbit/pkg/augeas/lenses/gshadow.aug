(*
 Module: Gshadow
 Parses /etc/gshadow

 Author: Lorenzo M. Catucci <catucci@ccd.uniroma2.it>

 Original Author: Free Ekanayaka <free@64studio.com>

 About: Reference
   - man 5 gshadow

 About: License
   This file is licensed under the LGPL v2+, like the rest of Augeas.

 About:

 Each line in the gshadow files represents the additional shadow-defined
 attributes for the corresponding group, as defined in the group file.

*)

module Gshadow =

   autoload xfm

(************************************************************************
 *                           USEFUL PRIMITIVES
 *************************************************************************)

let eol        = Util.eol
let comment    = Util.comment
let empty      = Util.empty

let colon      = Sep.colon
let comma      = Sep.comma

let sto_to_spc = store Rx.space_in

let word    = Rx.word
let password = /[A-Za-z0-9_.!*-]*/
let integer = Rx.integer

(************************************************************************
 * Group:                        ENTRIES
 *************************************************************************)

(* View: member *)
let member       = [ label "member" . store word ]
(* View: member_list
         the member list is a comma separated list of
         users allowed to chgrp to the group without
         being prompted for the group's password *)
let member_list  = Build.opt_list member comma

(* View: admin *)
let admin      = [ label "admin" . store word ]
(* View: admin_list
         the admin_list is a comma separated list of
         users allowed to change the group's password
         and the member_list *)
let admin_list = Build.opt_list admin comma

(* View: params *)
let params     = [ label "password"  . store password  . colon ]
		 .  admin_list?     . colon
                 .  member_list?

let entry      = Build.key_value_line word colon params

(************************************************************************
 *                                LENS
 *************************************************************************)

let lns        = (comment|empty|entry) *

let filter
               = incl "/etc/gshadow"
               . Util.stdexcl

let xfm        = transform lns filter
