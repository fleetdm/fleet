(*
Module: Resolv
  Parses /etc/resolv.conf

Author: Raphael Pinson <raphink@gmail.com>

About: Reference
  This lens tries to keep as close as possible to `man resolv.conf` where possible.

About: Licence
  This file is licensed under the LGPL v2+, like the rest of Augeas.

About: Lens Usage

About: Configuration files
  This lens applies to /etc/resolv.conf. See <filter>.
*)

module Resolv =
autoload xfm

(************************************************************************
 * Group:                 USEFUL PRIMITIVES
 *************************************************************************)

(* View: comment *)
let comment = Util.comment_generic /[ \t]*[;#][ \t]*/ "# "

(* View: comment_eol *)
let comment_eol = Util.comment_generic /[ \t]*[;#][ \t]*/ " # "

(* View: empty *)
let empty = Util.empty_generic_dos /[ \t]*[#;]?[ \t]*/


(************************************************************************
 * Group:                 MAIN OPTIONS
 *************************************************************************)

(* View: netmask
A network mask for IP addresses *)
let netmask = [ label "netmask" . Util.del_str "/" . store Rx.ip ]

(* View: ipaddr
An IP address or range with an optional mask *)
let ipaddr = [label "ipaddr" . store Rx.ip . netmask?]


(* View: nameserver
     A nameserver entry *)
let nameserver = Build.key_value_line_comment
                    "nameserver" Sep.space (store Rx.ip) comment_eol

(* View: domain *)
let domain = Build.key_value_line_comment
                    "domain" Sep.space (store Rx.word) comment_eol

(* View: search *)
let search = Build.key_value_line_comment
                    "search" Sep.space
                    (Build.opt_list
                           [label "domain" . store Rx.word]
                            Sep.space)
                    comment_eol

(* View: sortlist *)
let sortlist = Build.key_value_line_comment
                    "sortlist" Sep.space
                    (Build.opt_list
                           ipaddr
                           Sep.space)
                    comment_eol

(* View: lookup *)
let lookup =
  let lookup_entry = Build.flag("bind"|"file"|"yp")
    in Build.key_value_line_comment
             "lookup" Sep.space
             (Build.opt_list
                    lookup_entry
                    Sep.space)
             comment_eol

(* View: family *)
let family =
  let family_entry = Build.flag("inet4"|"inet6")
    in Build.key_value_line_comment
             "family" Sep.space
             (Build.opt_list
                    family_entry
                    Sep.space)
             comment_eol

(************************************************************************
 * Group:                 SPECIAL OPTIONS
 *************************************************************************)

(* View: ip6_dotint
     ip6-dotint option, which supports negation *)
let ip6_dotint =
  let negate = [ del "no-" "no-" . label "negate" ]
    in [ negate? . key "ip6-dotint" ]

(* View: options
     Options values *)
let options =
      let options_entry = Build.key_value ("ndots"|"timeout"|"attempts")
                                          (Util.del_str ":") (store Rx.integer)
                        | Build.flag ("debug"|"rotate"|"no-check-names"
                                     |"inet6"|"ip6-bytestring"|"edns0"
                                     |"single-request"|"single-request-reopen"
                                     |"no-tld-query"|"use-vc"|"no-reload")
                        | ip6_dotint

            in Build.key_value_line_comment
                    "options" Sep.space
                    (Build.opt_list
                           options_entry
                           Sep.space)
                    comment_eol

(* View: entry *)
let entry = nameserver
          | domain
          | search
          | sortlist
          | options
          | lookup
          | family

(* View: lns *)
let lns = ( empty | comment | entry )*

(* Variable: filter *)
let filter = (incl "/etc/resolv.conf")

let xfm = transform lns filter
