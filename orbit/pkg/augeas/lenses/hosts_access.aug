(*
Module: Hosts_Access
  Parses /etc/hosts.{allow,deny}

Author: Raphael Pinson <raphink@gmail.com>

About: Reference
  This lens tries to keep as close as possible to `man 5 hosts_access` and `man 5 hosts_options` where possible.

About: License
   This file is licenced under the LGPL v2+, like the rest of Augeas.

About: Lens Usage
   To be documented

About: Configuration files
   This lens applies to /etc/hosts.{allow,deny}. See <filter>.
*)

module Hosts_Access =

autoload xfm

(************************************************************************
 * Group:                 USEFUL PRIMITIVES
 *************************************************************************)

(* View: colon *)
let colon = del /[ \t]*(\\\\[ \t]*\n[ \t]+)?:[ \t]*(\\\\[ \t]*\n[ \t]+)?/ ": "

(* Variable: comma_sep *)
let comma_sep = /([ \t]|(\\\\\n))*,([ \t]|(\\\\\n))*/

(* Variable: ws_sep *)
let ws_sep = / +/

(* View: list_sep *)
let list_sep = del ( comma_sep | ws_sep ) ", "

(* View: list_item *)
let list_item = store ( Rx.word - /EXCEPT/i )

(* View: client_host_item
   Allows @ for netgroups, supports [ipv6] syntax *)
let client_host_item =
  let client_hostname_rx = /[A-Za-z0-9_.@?*-][A-Za-z0-9_.?*-]*/ in
  let client_ipv6_rx = "[" . /[A-Za-z0-9:?*%]+/ . "]" in
    let client_host_rx = client_hostname_rx | client_ipv6_rx in
    let netmask = [ Util.del_str "/" . label "netmask" . store Rx.word ] in
      store ( client_host_rx - /EXCEPT/i ) . netmask?

(* View: client_file_item *)
let client_file_item =
  let client_file_rx = /\/[^ \t\n,:]+/ in
    store ( client_file_rx - /EXCEPT/i )

(* Variable: option_kw
   Since either an option or a shell command can be given, use an explicit list
   of known options to avoid misinterpreting a command as an option *)
let option_kw = "severity"
              | "spawn"
              | "twist"
              | "keepalive"
              | "linger"
              | "rfc931"
              | "banners"
              | "nice"
              | "setenv"
              | "umask"
              | "user"
              | /allow/i
              | /deny/i

(* Variable: shell_command_rx *)
let shell_command_rx = /[^ \t\n:][^\n]*[^ \t\n]|[^ \t\n:\\\\]/
                         - ( option_kw . /.*/ )

(* View: sto_to_colon
   Allows escaped colon sequences *)
let sto_to_colon = store /[^ \t\n:=][^\n:]*((\\\\:|\\\\[ \t]*\n[ \t]+)[^\n:]*)*[^ \\\t\n:]|[^ \t\n:\\\\]/

(* View: except
 * The except operator makes it possible to write very compact rules.
 *)
let except (lns:lens) = [ label "except" . Sep.space
                        . del /except/i "EXCEPT"
                        . Sep.space . lns ]

(************************************************************************
 * Group:                 ENTRY TYPES
 *************************************************************************)

(* View: daemon *)
let daemon =
  let host = [ label "host"
             . Util.del_str "@"
             . list_item ] in
   [ label "process"
   . list_item
   . host? ]

(* View: daemon_list
    A list of <daemon>s *)
let daemon_list = Build.opt_list daemon list_sep

(* View: client *)
let client =
  let user = [ label "user"
             . list_item
             . Util.del_str "@" ] in
    [ label "client"
    . user?
    . client_host_item ]

(* View: client_file *)
let client_file = [ label "file" . client_file_item ]

(* View: client_list
    A list of <client>s *)
let client_list = Build.opt_list ( client | client_file ) list_sep

(* View: option
   Optional extensions defined in hosts_options(5) *)
let option = [ key option_kw
             . ( del /([ \t]*=[ \t]*|[ \t]+)/ " " . sto_to_colon )? ]

(* View: shell_command *)
let shell_command = [ label "shell_command"
                    . store shell_command_rx ]

(* View: entry *)
let entry = [ seq "line"
            . daemon_list
            . (except daemon_list)?
            . colon
            . client_list
            . (except client_list)?
            . ( (colon . option)+ | (colon . shell_command)? )
            . Util.eol ]

(************************************************************************
 * Group:                 LENS AND FILTER
 *************************************************************************)

(* View: lns *)
let lns = (Util.empty | Util.comment | entry)*

(* View: filter *)
let filter = incl "/etc/hosts.allow"
           . incl "/etc/hosts.deny"

let xfm = transform lns filter
