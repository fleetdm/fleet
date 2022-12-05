(*
Module: Keepalived
  Parses /etc/keepalived/keepalived.conf

Author: Raphael Pinson <raphink@gmail.com>

About: Reference
  This lens tries to keep as close as possible to `man 5 keepalived.conf` where possible.

About: License
   This file is licenced under the LGPL v2+, like the rest of Augeas.

About: Lens Usage
   To be documented

About: Configuration files
   This lens applies to /etc/keepalived/keepalived.conf. See <filter>.

About: Examples
   The <Test_Keepalived> file contains various examples and tests.
*)


module Keepalived =
  autoload xfm

(************************************************************************
 * Group:                 USEFUL PRIMITIVES
 *************************************************************************)

(* Group: Comments and empty lines *)

(* View: indent *)
let indent = Util.indent

(* View: eol *)
let eol = Util.eol

(* View: opt_eol *)
let opt_eol = del /[ \t]*\n?/ " "

(* View: sep_spc *)
let sep_spc = Sep.space

(* View: comment
Map comments in "#comment" nodes *)
let comment = Util.comment_generic /[ \t]*[#!][ \t]*/ "# "

(* View: comment_eol
Map comments at eol *)
let comment_eol = Util.comment_generic /[ \t]*[#!][ \t]*/ " # "

(* View: comment_or_eol
A <comment_eol> or <eol> *)
let comment_or_eol = comment_eol | (del /[ \t]*[#!]?\n/ "\n")

(* View: empty
Map empty lines *)
let empty   = Util.empty

(* View: sto_email_addr *)
let sto_email_addr = store Rx.email_addr

(* Variable: word *)
let word = Rx.word

(* Variable: word_slash *)
let word_slash = word | "/"

(* View: sto_word *)
let sto_word = store word

(* View: sto_num *)
let sto_num = store Rx.relinteger

(* View: sto_ipv6 *)
let sto_ipv6 = store Rx.ipv6

(* View: sto_to_eol *)
let sto_to_eol = store /[^#! \t\n][^#!\n]*[^#! \t\n]|[^#! \t\n]/

(* View: field *)
let field (kw:regexp) (sto:lens) = indent . Build.key_value_line_comment kw sep_spc sto comment_eol

(* View: flag
A single word *)
let flag (kw:regexp) = [ indent . key kw . comment_or_eol ]

(* View: ip_port
   An IP <space> port pair *)
let ip_port = [ label "ip" . sto_word ] . sep_spc . [ label "port" . sto_num ]

(* View: lens_block
A generic block with a title lens.
The definition is very similar to Build.block_newlines
but uses a different type of <comment>. *)
let lens_block (title:lens) (sto:lens) =
   [ indent . title
   . Build.block_newlines sto comment . eol ]

(* View: block
A simple block with just a block title *)
let block (kw:regexp) (sto:lens) = lens_block (key kw) sto

(* View: named_block
A block with a block title and name *)
let named_block (kw:string) (sto:lens) = lens_block (key kw . sep_spc . sto_word) sto

(* View: named_block_arg_title
A title lens for named_block_arg *)
let named_block_arg_title (kw:string) (name:string) (arg:string) =
                            key kw . sep_spc
                          . [ label name . sto_word ]
                          . sep_spc
                          . [ label arg . sto_word ]

(* View: named_block_arg
A block with a block title, a name and an argument *)
let named_block_arg (kw:string) (name:string) (arg:string) (sto:lens) =
                           lens_block (named_block_arg_title kw name arg) sto


(************************************************************************
 * Group:                 GLOBAL CONFIGURATION
 *************************************************************************)

(* View: email
A simple email address entry *)
let email = [ indent . label "email" . sto_email_addr . comment_or_eol ]

(* View: global_defs_field
Possible fields in the global_defs block *)
let global_defs_field =
      let word_re = "smtp_server"|"lvs_id"|"router_id"|"vrrp_mcast_group4"
   in let ipv6_re = "vrrp_mcast_group6"
   in let num_re = "smtp_connect_timeout"
   in block "notification_email" email
    | field "notification_email_from" sto_email_addr
    | field word_re sto_word
    | field num_re sto_num
    | field ipv6_re sto_ipv6

(* View: global_defs
A global_defs block *)
let global_defs = block "global_defs" global_defs_field

(* View: prefixlen
A prefix for IP addresses *)
let prefixlen = [ label "prefixlen" . Util.del_str "/" . sto_num ]

(* View: ipaddr
An IP address or range with an optional mask *)
let ipaddr = label "ipaddr" . store /[0-9.-]+/ . prefixlen?

(* View: ipdev
A device for IP addresses *)
let ipdev = [ key "dev" . sep_spc . sto_word ]

(* View: static_ipaddress_field
The whole string is fed to ip addr add.
You can truncate the string anywhere you like and let ip addr add use defaults for the rest of the string.
To be refined with fields according to `ip addr help`.
*)
let static_ipaddress_field = [ indent . ipaddr
                             . (sep_spc . ipdev)?
                             . comment_or_eol ]

(* View: static_routes_field
src $SRC_IP to $DST_IP dev $SRC_DEVICE
*)
let static_routes_field = [ indent . label "route"
                          . [ key "src" . sto_word ] . sep_spc
                          . [ key "to"  . sto_word ] . sep_spc
                          . [ key "dev" . sto_word ] . comment_or_eol ]

(* View: static_routes *)
let static_routes = block "static_ipaddress" static_ipaddress_field
                  | block "static_routes" static_routes_field


(* View: global_conf
A global configuration entry *)
let global_conf = global_defs | static_routes


(************************************************************************
 * Group:                 VRRP CONFIGURATION
 *************************************************************************)

(*View: vrrp_sync_group_field *)
let vrrp_sync_group_field =
      let to_eol_re = /notify(_master|_backup|_fault|_stop|_deleted)?/
   in let flag_re = "smtp_alert"
   in field to_eol_re sto_to_eol
    | flag flag_re
    | block "group" [ indent . key word . comment_or_eol ]

(* View: vrrp_sync_group *)
let vrrp_sync_group = named_block "vrrp_sync_group" vrrp_sync_group_field

(* View: vrrp_instance_field *)
let vrrp_instance_field =
      let word_re = "state" | "interface" | "lvs_sync_daemon_interface"
   in let num_re = "virtual_router_id" | "priority" | "advert_int" | /garp_master_(delay|repeat|refresh|refresh_repeat)/
   in let to_eol_re = /notify(_master|_backup|_fault|_stop|_deleted)?/ | /(mcast|unicast)_src_ip/
   in let flag_re = "smtp_alert" | "nopreempt" | "ha_suspend" | "debug" | "use_vmac" | "vmac_xmit_base" | "native_ipv6" | "dont_track_primary" | "preempt_delay"
   in field word_re sto_word
    | field num_re sto_num
    | field to_eol_re sto_to_eol
    | flag flag_re
    | block "authentication" (
         field /auth_(type|pass)/ sto_word
         )
    | block "virtual_ipaddress" static_ipaddress_field
    | block /track_(interface|script)/ ( flag word )
    | block "unicast_peer" static_ipaddress_field

(* View: vrrp_instance *)
let vrrp_instance = named_block "vrrp_instance" vrrp_instance_field

(* View: vrrp_script_field *)
let vrrp_script_field =
      let num_re = "interval" | "weight" | "fall" | "raise"
   in let to_eol_re = "script"
   in field to_eol_re sto_to_eol
    | field num_re sto_num

(* View: vrrp_script *)
let vrrp_script = named_block "vrrp_script" vrrp_script_field


(* View: vrrpd_conf
contains subblocks of VRRP synchronization group(s) and VRRP instance(s) *)
let vrrpd_conf = vrrp_sync_group | vrrp_instance | vrrp_script


(************************************************************************
 * Group:                 REAL SERVER CHECKS CONFIGURATION
 *************************************************************************)

(* View: tcp_check_field *)
let tcp_check_field =
      let word_re = "bindto"
   in let num_re = /connect_(timeout|port)/
   in field word_re sto_word
    | field num_re sto_num

(* View: misc_check_field *)
let misc_check_field =
      let flag_re = "misc_dynamic"
   in let num_re = "misc_timeout"
   in let to_eol_re = "misc_path"
   in field num_re sto_num
    | flag flag_re
    | field to_eol_re sto_to_eol

(* View: smtp_host_check_field *)
let smtp_host_check_field =
      let word_re = "connect_ip" | "bindto"
   in let num_re = "connect_port"
   in field word_re sto_word
    | field num_re sto_num

(* View: smtp_check_field *)
let smtp_check_field =
      let word_re = "connect_ip" | "bindto"
   in let num_re = "connect_timeout" | "retry" | "delay_before_retry"
   in let to_eol_re = "helo_name"
   in field word_re sto_word
    | field num_re sto_num
    | field to_eol_re sto_to_eol
    | block "host" smtp_host_check_field

(* View: http_url_check_field *)
let http_url_check_field =
      let word_re = "digest"
   in let num_re = "status_code"
   in let to_eol_re = "path"
   in field word_re sto_word
    | field num_re sto_num
    | field to_eol_re sto_to_eol

(* View: http_check_field *)
let http_check_field =
      let num_re = /connect_(timeout|port)/ | "nb_get_retry" | "delay_before_retry"
   in field num_re sto_num
    | block "url" http_url_check_field

(* View: real_server_field *)
let real_server_field =
      let num_re = "weight"
   in let flag_re = "inhibit_on_failure"
   in let to_eol_re = /notify_(up|down)/
   in field num_re sto_num
    | flag flag_re
    | field to_eol_re sto_to_eol
    | block "TCP_CHECK" tcp_check_field
    | block "MISC_CHECK" misc_check_field
    | block "SMTP_CHECK" smtp_check_field
    | block /(HTTP|SSL)_GET/ http_check_field

(************************************************************************
 * Group:                 LVS CONFIGURATION
 *************************************************************************)

(* View: virtual_server_field *)
let virtual_server_field =
      let num_re = "delay_loop" | "persistence_timeout" | "quorum" | "hysteresis"
   in let word_re = /lb_(algo|kind)/ | "nat_mask" | "protocol" | "persistence_granularity"
                      | "virtualhost"
   in let flag_re = "ops" | "ha_suspend" | "alpha" | "omega"
   in let to_eol_re = /quorum_(up|down)/
   in let ip_port_re = "sorry_server"
   in field num_re sto_num
    | field word_re sto_word
    | flag flag_re
    | field to_eol_re sto_to_eol
    | field ip_port_re ip_port
    | named_block_arg "real_server" "ip" "port" real_server_field

(* View: virtual_server *)
let virtual_server = named_block_arg "virtual_server" "ip" "port" virtual_server_field

(* View: virtual_server_group_field *)
let virtual_server_group_field = [ indent . label "vip"
                               . [ ipaddr ]
			       . sep_spc
			       . [ label "port" . sto_num ]
			       . comment_or_eol ]

(* View: virtual_server_group *)
let virtual_server_group = named_block "virtual_server_group" virtual_server_group_field

(* View: lvs_conf
contains subblocks of Virtual server group(s) and Virtual server(s) *)
let lvs_conf = virtual_server | virtual_server_group


(* View: lns
     The keepalived lens
*)
let lns = ( empty | comment | global_conf | vrrpd_conf | lvs_conf )*

(* Variable: filter *)
let filter = incl "/etc/keepalived/keepalived.conf"

let xfm = transform lns filter

