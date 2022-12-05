(*
Module: Pagekite
  Parses /etc/pagekite.d/

Author: Michael Pimmer <blubb@fonfon.at>

About: License
   This file is licenced under the LGPL v2+, like the rest of Augeas.
*)

module Pagekite =
autoload xfm

(* View: lns *)

(* Variables *)
let equals = del /[ \t]*=[ \t]*/ "="
let neg2 = /[^# \n\t]+/
let neg3 = /[^# \:\n\t]+/
let eol = del /\n/ "\n"
(* Match everything from here to eol, cropping whitespace at both ends *)
let to_eol  = /[^ \t\n](.*[^ \t\n])?/

(* A key followed by comma-separated values 
  k: name of the key
  key_sep: separator between key and values
  value_sep: separator between values
  sto: store for values
*)
let key_csv_line (k:string) (key_sep:lens) (value_sep:lens) (sto:lens) = 
  [ key k . key_sep . [ seq k . sto ] .
    [ seq k . value_sep . sto ]* . Util.eol ]

(* entries for pagekite.d/10_account.rc *)
let domain = [ key "domain" . equals . store neg2 . Util.comment_or_eol ]
let frontend = Build.key_value_line ("frontend" | "frontends") 
                                       equals (store Rx.neg1)
let host = Build.key_value_line "host" equals (store Rx.ip)
let ports = key_csv_line "ports" equals Sep.comma (store Rx.integer)
let protos = key_csv_line "protos" equals Sep.comma (store Rx.word)

(* entries for pagekite.d/20_frontends.rc *)
let kitesecret = Build.key_value_line "kitesecret" equals (store Rx.space_in)
let kv_frontend = Build.key_value_line ( "kitename" | "fe_certname" | 
                                         "ca_certs" | "tls_endpoint" ) 
                                       equals (store Rx.neg1)

(* entries for services like 80_httpd.rc *)
let service_colon = del /[ \t]*:[ \t]*/ " : "
let service_on = [ key "service_on" . [ seq "service_on" . equals .
                   [ label "protocol" . store neg3 ] . service_colon .
                   [ label "kitename" . (store neg3) ] . service_colon .
                   [ label "backend_host" . (store neg3) ] . service_colon .
                   [ label "backend_port" . (store neg3) ] . service_colon . (
                     [ label "secret" . (store Rx.no_spaces) . Util.eol ] | eol
                   ) ] ]

let service_cfg = [ key "service_cfg" . equals . store to_eol . eol ]

let flags = ( "defaults" | "isfrontend" | "abort_not_configured" | "insecure" )

let entries = Build.flag_line flags
        | domain
        | frontend
        | host
        | ports
        | protos
        | kv_frontend
        | kitesecret
        | service_on
        | service_cfg

let lns = ( entries | Util.empty | Util.comment )*

(* View: filter *)
let filter = incl "/etc/pagekite.d/*.rc"
        . Util.stdexcl

let xfm = transform lns filter
