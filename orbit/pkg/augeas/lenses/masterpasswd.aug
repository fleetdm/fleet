(*
 Module: MasterPasswd
 Parses /etc/master.passwd

 Author: Matt Dainty <matt@bodgit-n-scarper.com>

 About: Reference
        - man 5 master.passwd

 Each line in the master.passwd file represents a single user record, whose
 colon-separated attributes correspond to the members of the passwd struct

*)

module MasterPasswd =

   autoload xfm

(************************************************************************
 * Group:                    USEFUL PRIMITIVES
 *************************************************************************)

(* Group: Comments and empty lines *)

let eol        = Util.eol
let comment    = Util.comment
let empty      = Util.empty
let dels       = Util.del_str

let word       = Rx.word
let integer    = Rx.integer

let colon      = Sep.colon

let sto_to_eol = Passwd.sto_to_eol
let sto_to_col = Passwd.sto_to_col
(* Store an empty string if nothing matches *)
let sto_to_col_or_empty = Passwd.sto_to_col_or_empty

(************************************************************************
 * Group:                        ENTRIES
 *************************************************************************)

let username    = /[_.A-Za-z0-9][-_.A-Za-z0-9]*\$?/

(* View: password
        pw_passwd *)
let password    = [ label "password"    . sto_to_col?   . colon ]

(* View: uid
        pw_uid *)
let uid         = [ label "uid"         . store integer . colon ]

(* View: gid
        pw_gid *)
let gid         = [ label "gid"         . store integer . colon ]

(* View: class
        pw_class *)
let class       = [ label "class"       . sto_to_col? . colon ]

(* View: change
        pw_change *)
let change_date = [ label "change_date" . store integer? . colon ]

(* View: expire
        pw_expire *)
let expire_date = [ label "expire_date" . store integer? . colon ]

(* View: name
        pw_gecos; the user's full name *)
let name        = [ label "name"        . sto_to_col? . colon ]

(* View: home
        pw_dir *)
let home        = [ label "home"        . sto_to_col?   . colon ]

(* View: shell
        pw_shell *)
let shell       = [ label "shell"       . sto_to_eol? ]

(* View: entry
        struct passwd *)
let entry       = [ key username
                . colon
                . password
                . uid
                . gid
                . class
                . change_date
                . expire_date
                . name
                . home
                . shell
                . eol ]

(* NIS entries *)
let niscommon   =  [ label "password"    . sto_to_col ]?    . colon
               . [ label "uid"         . store integer ]? . colon
               . [ label "gid"         . store integer ]? . colon
               . [ label "class"       . sto_to_col ]?    . colon
               . [ label "change_date" . store integer ]? . colon
               . [ label "expire_date" . store integer ]? . colon
               . [ label "name"        . sto_to_col ]?    . colon
               . [ label "home"        . sto_to_col ]?    . colon
               . [ label "shell"       . sto_to_eol ]?

let nisentry =
  let overrides =
        colon
      . niscommon in
  [ dels "+@" . label "@nis" . store username . overrides . eol ]

let nisuserplus =
  let overrides =
        colon
      . niscommon in
  [ dels "+" . label "@+nisuser" . store username . overrides . eol ]

let nisuserminus =
  let overrides =
        colon
      . niscommon in
  [ dels "-" . label "@-nisuser" . store username . overrides . eol ]

let nisdefault =
  let overrides =
        colon
      . [ label "password"    . sto_to_col_or_empty . colon ]
      . [ label "uid"         . store integer? . colon ]
      . [ label "gid"         . store integer? . colon ]
      . [ label "class"       . sto_to_col?    . colon ]
      . [ label "change_date" . store integer? . colon ]
      . [ label "expire_date" . store integer? . colon ]
      . [ label "name"        . sto_to_col?    . colon ]
      . [ label "home"        . sto_to_col?    . colon ]
      . [ label "shell"       . sto_to_eol? ] in
  [ dels "+" . label "@nisdefault" . overrides? . eol ]

(************************************************************************
 *                                LENS
 *************************************************************************)

let lns        = (comment|empty|entry|nisentry|nisdefault|nisuserplus|nisuserminus) *

let filter     = incl "/etc/master.passwd"

let xfm        = transform lns filter
