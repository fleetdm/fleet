(*
Module: Rabbitmq
  Parses Rabbitmq configuration files

Author: Raphael Pinson <raphael.pinson@camptocamp.com>

About: Reference
  This lens tries to keep as close as possible to `http://www.rabbitmq.com/configure.html` where possible.

About: License
   This file is licenced under the LGPL v2+, like the rest of Augeas.

About: Lens Usage
   To be documented

About: Configuration files
   This lens applies to Rabbitmq configuration files. See <filter>.

About: Examples
   The <Test_Rabbitmq> file contains various examples and tests.
*)
module Rabbitmq =

autoload xfm

(* View: listeners
     A tcp/ssl listener *)
let listeners =
     let value = Erlang.make_value Erlang.integer
               | Erlang.tuple Erlang.quoted Erlang.integer
  in Erlang.list /(tcp|ssl)_listeners/ value


(* View: ssl_options
    (Incomplete) list of SSL options *)
let ssl_options =
     let versions_list = Erlang.opt_list (Erlang.make_value Erlang.quoted)
  in let option = Erlang.value /((ca)?cert|key)file/ Erlang.path
                | Erlang.value "verify" Erlang.bare
                | Erlang.value "verify_fun" Erlang.boolean
                | Erlang.value /fail_if_no_peer_cert|reuse_sessions/ Erlang.boolean
                | Erlang.value "depth" Erlang.integer
                | Erlang.value "password" Erlang.quoted
                | Erlang.value "versions" versions_list
  in Erlang.list "ssl_options" option

(* View: disk_free_limit *)
let disk_free_limit =
     let value = Erlang.integer | Erlang.tuple Erlang.bare Erlang.decimal
  in Erlang.value "disk_free_limit" value

(* View: log_levels *)
let log_levels =
     let category = Erlang.tuple Erlang.bare Erlang.bare
  in Erlang.list "log_levels" category

(* View: cluster_nodes
     Can be a tuple `(nodes, node_type)` or simple `nodes` *)
let cluster_nodes =
     let nodes = Erlang.opt_list (Erlang.make_value Erlang.quoted)
  in let value = Erlang.tuple nodes Erlang.bare
               | nodes
  in Erlang.value "cluster_nodes" value

(* View: cluster_partition_handling
     Can be single value or
     `{pause_if_all_down, [nodes], ignore | autoheal}` *)
let cluster_partition_handling =
     let nodes = Erlang.opt_list (Erlang.make_value Erlang.quoted)
  in let value = Erlang.tuple3 Erlang.bare nodes Erlang.bare
               | Erlang.bare
  in Erlang.value "cluster_partition_handling" value

(* View: tcp_listen_options *)
let tcp_listen_options =
     let value = Erlang.make_value Erlang.bare
               | Erlang.tuple Erlang.bare Erlang.bare
  in Erlang.list "tcp_listen_options" value

(* View: parameters
     Top-level parameters for the lens *)
let parameters = listeners
               | ssl_options
               | disk_free_limit
               | log_levels
               | Erlang.value /vm_memory_high_watermark(_paging_ratio)?/ Erlang.decimal
               | Erlang.value "frame_max" Erlang.integer
               | Erlang.value "heartbeat" Erlang.integer
               | Erlang.value /default_(vhost|user|pass)/ Erlang.glob
               | Erlang.value_list "default_user_tags" Erlang.bare
               | Erlang.value_list "default_permissions" Erlang.glob
               | cluster_nodes
               | Erlang.value_list "server_properties" Erlang.bare
               | Erlang.value "collect_statistics" Erlang.bare
               | Erlang.value "collect_statistics_interval" Erlang.integer
               | Erlang.value_list "auth_mechanisms" Erlang.quoted
               | Erlang.value_list "auth_backends" Erlang.bare
               | Erlang.value "delegate_count" Erlang.integer
               | Erlang.value_list "trace_vhosts" Erlang.bare
               | tcp_listen_options
               | Erlang.value "hipe_compile" Erlang.boolean
               | Erlang.value "msg_store_index_module" Erlang.bare
               | Erlang.value "backing_queue_module" Erlang.bare
               | Erlang.value "msg_store_file_size_limit" Erlang.integer
               | Erlang.value /queue_index_(max_journal_entries|embed_msgs_below)/ Erlang.integer
               | cluster_partition_handling
               | Erlang.value /(ssl_)?handshake_timeout/ Erlang.integer
               | Erlang.value "channel_max" Erlang.integer
               | Erlang.value_list "loopback_users" Erlang.glob
               | Erlang.value "reverse_dns_lookups" Erlang.boolean
               | Erlang.value "cluster_keepalive_interval" Erlang.integer
               | Erlang.value "mnesia_table_loading_timeout" Erlang.integer

(* View: rabbit
    The rabbit <Erlang.application> config *)
let rabbit = Erlang.application "rabbit" parameters

(* View: lns
    A top-level <Erlang.config> *)
let lns = Erlang.config rabbit

(* Variable: filter *)
let filter = incl "/etc/rabbitmq/rabbitmq.config"

let xfm = transform lns filter
