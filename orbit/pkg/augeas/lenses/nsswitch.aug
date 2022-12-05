(*
Module: Nsswitch
  Parses /etc/nsswitch.conf

Author: Raphael Pinson <raphink@gmail.com>

About: Reference
  This lens tries to keep as close as possible to `man nsswitch.conf` where possible.

About: Licence
  This file is licensed under the LGPL v2+, like the rest of Augeas.

About: Lens Usage

About: Configuration files
  This lens applies to /etc/nsswitch.conf. See <filter>.
*)

module Nsswitch =
autoload xfm

(************************************************************************
 * Group:                 USEFUL PRIMITIVES
 *************************************************************************)

(* View: comment *)
let comment = Util.comment

(* View: empty *)
let empty = Util.empty

(* View: sep_colon
    The separator for database entries *)
let sep_colon = del /:[ \t]*/ ": "

(* View: database_kw
    The database specification like `passwd', `shadow', or `hosts' *)
let database_kw = Rx.word

(* View: service
    The service specification like `files', `db', or `nis' *)
let service = [ label "service" . store Rx.word ]

(* View: reaction
    The reaction on lookup result like `[NOTFOUND=return]'
    TODO: Use case-insensitive regexps when ticket #147 is fixed.
*)
let reaction =
  let status_kw = /[Ss][Uu][Cc][Cc][Ee][Ss][Ss]/
                | /[Nn][Oo][Tt][Ff][Oo][Uu][Nn][Dd]/
                | /[Uu][Nn][Aa][Vv][Aa][Ii][Ll]/
                | /[Tt][Rr][Yy][Aa][Gg][Aa][Ii][Nn]/
    in let action_kw = /[Rr][Ee][Tt][Uu][Rr][Nn]/
                     | /[Cc][Oo][Nn][Tt][Ii][Nn][Uu][Ee]/
                     | /[Mm][Ee][Rr][Gg][Ee]/
      in let negate = [ Util.del_str "!" . label "negate" ]
        in let reaction_entry = [ label "status" . negate?
                                . store status_kw
                                . Util.del_str "="
                                . [ label "action" . store action_kw ] ]
          in Util.del_str "["
             . [ label "reaction"
               . (Build.opt_list reaction_entry Sep.space) ]
             . Util.del_str "]"

(* View: database *)
let database = 
    [ label "database" . store database_kw
       . sep_colon
       . (Build.opt_list
            (service|reaction)
            Sep.space)
       . Util.comment_or_eol ]

(* View: lns *)
let lns = ( empty | comment | database )*

(* Variable: filter *)
let filter = (incl "/etc/nsswitch.conf")

let xfm = transform lns filter
