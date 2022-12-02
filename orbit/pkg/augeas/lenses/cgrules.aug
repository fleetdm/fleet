(*
Module: cgrules
    Parses /etc/cgrules.conf

Author:
    Raphael Pinson          <raphink@gmail.com>
    Ivana Hutarova Varekova <varekova@redhat.com>

About: Licence
    This file is licensed under the LGPL v2+, like the rest of Augeas.

About: Lens Usage
    Sample usage of this lens in augtool:

About: Configuration files
   This lens applies to /etc/cgconfig.conf. See <filter>.
 *)

module Cgrules =
   autoload xfm

(************************************************************************
 * Group:                 USEFUL PRIMITIVES
 *************************************************************************)

(* Group: Separators *)
(* Variable: ws *)
   let ws = del /[ \t]+/ " "

(* Group: Comments and empty lines *)
(* Variable: eol *)
   let eol     = Util.eol

(* Variable: comment *)
   let comment = Util.comment

(* Variable: empty *)
   let empty   = Util.empty

(* Group: Generic primitive definitions *)
(* Variable: name *)
   let name       = /[^@%# \t\n][^ \t\n]*/
(* Variable: ctrl_key *)
   let ctrl_key   = /[^ \t\n\/]+/
(* Variable: ctrl_value *)
   let ctrl_value = /[^ \t\n]+/

(************************************************************************
 * Group:                 CONTROLLER
 *************************************************************************)

(* Variable: controller *)
let controller =  ws . [ key ctrl_key . ws . store ctrl_value ]

let more_controller = Util.del_str "%" . controller . eol

(************************************************************************
 * Group:                 RECORDS
 *************************************************************************)

let generic_record (lbl:string) (lns:lens) =
                  [ label lbl . lns
                  . controller . eol
                  . more_controller* ]

(* Variable: user_record *)
let user_record = generic_record "user" (store name)

(* Variable: group_record *)
let group_record = generic_record "group" (Util.del_str "@" . store name)

(************************************************************************
 * Group:                        LENS & FILTER
 *************************************************************************)

(* View: lns
     The main lens, any amount of
       * <empty> lines
       * <comment>
       * <user_record>
       * <group_record>
*)
let lns =  (  empty | comment | user_record | group_record )*

let xfm = transform lns (incl "/etc/cgrules.conf")
