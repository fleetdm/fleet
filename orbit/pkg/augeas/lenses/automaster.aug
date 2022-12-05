(*
Module: Automaster
  Parses autofs' auto.master files

Author: Dominic Cleal <dcleal@redhat.com>

About: Reference
  See auto.master(5)

About: License
   This file is licenced under the LGPL v2+, like the rest of Augeas.

About: Lens Usage
   To be documented

About: Configuration files
   This lens applies to /etc/auto.master, auto_master and /etc/auto.master.d/*
   files.

About: Examples
   The <Test_Automaster> file contains various examples and tests.
*)

module Automaster =
autoload xfm

(************************************************************************
 * Group:                 USEFUL PRIMITIVES
 *************************************************************************)

(* View: eol *)
let eol = Util.eol

(* View: empty *)
let empty   = Util.empty

(* View: comment *)
let comment = Util.comment

(* View mount *)
let mount = /[^+ \t\n#]+/

(* View: type
   yp, file, dir etc but not ldap *)
let type = Rx.word - /ldap/

(* View: format
   sun, hesoid *)
let format = Rx.word

(* View: name *)
let name = /[^: \t\n]+/

(* View: host *)
let host = /[^:# \n\t]+/

(* View: dn *)
let dn = /[^:# \n\t]+/

(* An option label can't contain comma, comment, equals, or space *)
let optlabel = /[^,#= \n\t]+/
let spec    = /[^,# \n\t][^ \n\t]*/

(* View: optsep *)
let optsep = del /[ \t,]+/ ","

(************************************************************************
 * Group:                 ENTRIES
 *************************************************************************)

(* View: map_format *)
let map_format = [ label "format" . store format ]

(* View: map_type *)
let map_type = [ label "type" . store type ]

(* View: map_name *)
let map_name = [ label "map" . store name ]

(* View: map_generic
   Used for all except LDAP maps which are parsed further *)
let map_generic = ( map_type . ( Sep.comma . map_format )?  . Sep.colon )?
                    . map_name

(* View: map_ldap_name
   Split up host:dc=foo into host/map nodes *)
let map_ldap_name = ( [ label "host" . store host ] . Sep.colon )?
                      . [ label "map" . store dn ]

(* View: map_ldap *)
let map_ldap      = [ label "type" . store "ldap" ]
                      . ( Sep.comma . map_format )? . Sep.colon
                      . map_ldap_name

(* View: comma_spc_sep_list
   Parses options either for filesystems or autofs *)
let comma_spc_sep_list (l:string) =
  let value = [ label "value" . Util.del_str "=" . store Rx.neg1 ] in
    let lns = [ label l . store optlabel . value? ] in
       Build.opt_list lns optsep

(* View: map_mount 
   Mountpoint and whitespace, followed by the map info *)
let map_mount  = [ seq "map" . store mount . Util.del_ws_tab
                   . ( map_generic | map_ldap )
                   . ( Util.del_ws_spc . comma_spc_sep_list "opt" )?
                   . Util.eol ]

(* map_master
   "+" to include more master entries and optional whitespace *)
let map_master = [ seq "map" . store "+" . Util.del_opt_ws ""
                   . ( map_generic | map_ldap )
                   . ( Util.del_ws_spc . comma_spc_sep_list "opt" )?
                   . Util.eol ]

(* View: lns *)
let lns = ( empty | comment | map_mount | map_master ) *

(* Variable: filter *)
let filter = incl "/etc/auto.master"
           . incl "/etc/auto_master"
           . incl "/etc/auto.master.d/*"
           . Util.stdexcl

let xfm = transform lns filter
