(*
Module: Redis
  Parses Redis's configuration files

Author: Marc Fournier <marc.fournier@camptocamp.com>

About: Reference
    This lens is based on Redis's default redis.conf

About: Usage Example
(start code)
augtool> set /augeas/load/Redis/incl "/etc/redis/redis.conf"
augtool> set /augeas/load/Redis/lens "Redis.lns"
augtool> load

augtool> get /files/etc/redis/redis.conf/vm-enabled
/files/etc/redis/redis.conf/vm-enabled = no
augtool> print /files/etc/redis/redis.conf/rename-command[1]/
/files/etc/redis/redis.conf/rename-command
/files/etc/redis/redis.conf/rename-command/from = "CONFIG"
/files/etc/redis/redis.conf/rename-command/to = "CONFIG2"

augtool> set /files/etc/redis/redis.conf/activerehashing no
augtool> save
Saved 1 file(s)
augtool> set /files/etc/redis/redis.conf/save[1]/seconds 123
augtool> set /files/etc/redis/redis.conf/save[1]/keys 456
augtool> save
Saved 1 file(s)
(end code)
   The <Test_Redis> file also contains various examples.

About: License
  This file is licensed under the LGPL v2+, like the rest of Augeas.
*)

module Redis =
autoload xfm

let k = Rx.word
let v = /[^ \t\n'"]+/
let comment = Util.comment
let empty = Util.empty
let indent = Util.indent
let eol = Util.eol
let del_ws_spc = Util.del_ws_spc
let dquote = Util.del_str "\""

(* View: standard_entry
A standard entry is a key-value pair, separated by blank space, with optional
blank spaces at line beginning & end. The value part can be optionnaly enclosed
in single or double quotes. Comments at end-of-line ar NOT allowed by
redis-server.
*)
let standard_entry =
     let reserved_k = "save" | "rename-command" | "replicaof" | "slaveof"
                    | "bind" | "client-output-buffer-limit"
                    | "sentinel"
  in let entry_noempty = [ indent . key (k - reserved_k) . del_ws_spc
                         . Quote.do_quote_opt_nil (store v) . eol ]
  in let entry_empty = [ indent . key (k - reserved_k) . del_ws_spc
                         . dquote . store "" . dquote . eol ]
  in entry_noempty | entry_empty

let save = /save/
let seconds = [ label "seconds" . Quote.do_quote_opt_nil (store Rx.integer) ]
let keys = [ label "keys" . Quote.do_quote_opt_nil (store Rx.integer) ]
let save_val =
     let save_val_empty = del_ws_spc . dquote . store "" . dquote
  in let save_val_sec_keys = del_ws_spc . seconds . del_ws_spc . keys
  in save_val_sec_keys | save_val_empty

(* View: save_entry
Entries identified by the "save" keyword can be found more than once. They have
2 mandatory parameters, both integers or a single parameter that is empty double quoted
string. The same rules as standard_entry apply for quoting, comments and whitespaces.
*)
let save_entry = [ indent . key save . save_val . eol ]

let replicaof = /replicaof|slaveof/
let ip = [ label "ip" . Quote.do_quote_opt_nil (store Rx.ip) ]
let port = [ label "port" . Quote.do_quote_opt_nil (store Rx.integer) ]
(* View: replicaof_entry
Entries identified by the "replicaof" keyword can be found more than once. They
have 2 mandatory parameters, the 1st one is an IP address, the 2nd one is a
port number. The same rules as standard_entry apply for quoting, comments and
whitespaces.
*)
let replicaof_entry = [ indent . key replicaof . del_ws_spc . ip . del_ws_spc . port . eol ]

let sentinel_global_entry =
     let keys  = "deny-scripts-reconfig" | "current-epoch" | "myid"
  in store keys .
       del_ws_spc . [ label "value" . store ( Rx.word | Rx.integer ) ]

let sentinel_cluster_setup =
     let keys = "config-epoch" | "leader-epoch"
  in store keys .
       del_ws_spc . [ label "cluster" . store Rx.word ] .
       del_ws_spc . [ label "epoch" . store Rx.integer ]

let sentinel_cluster_instance_setup = 
     let keys = "monitor" | "known-replica"
  in store keys .
       del_ws_spc . [ label "cluster" . store Rx.word ] .
       del_ws_spc. [ label "ip" . store Rx.ip ] .
       del_ws_spc . [ label "port" . store Rx.integer ] .
       (del_ws_spc .  [ label "quorum" . store Rx.integer ])?

let sentinel_clustering =
     let keys = "known-sentinel"
  in store keys .
       del_ws_spc . [ label "cluster" . store Rx.word ] .
       del_ws_spc . [ label "ip" . store Rx.ip ] .
       del_ws_spc . [ label "port" . store Rx.integer ] .
       del_ws_spc . [ label "id" . store Rx.word ]

(* View: sentinel_entry
*)
let sentinel_entry =
  indent . [ key "sentinel" . del_ws_spc .
    (sentinel_global_entry | sentinel_cluster_setup | sentinel_cluster_instance_setup | sentinel_clustering)
  ] . eol

(* View: bind_entry
The "bind" entry can be passed one or several ip addresses. A bind
statement "bind ip1 ip2 .. ipn" results in a tree
{ "bind" { "ip" = ip1 } { "ip" = ip2 } ... { "ip" = ipn } }
*)
let bind_entry =
  let ip = del_ws_spc . Quote.do_quote_opt_nil (store Rx.ip) in
  indent . [ key "bind" . [ label "ip" . ip ]+ ] . eol

let renamecmd = /rename-command/
let from = [ label "from" . Quote.do_quote_opt_nil (store Rx.word) ]
let to = [ label "to" . Quote.do_quote_opt_nil (store Rx.word) ]
(* View: save_entry
Entries identified by the "rename-command" keyword can be found more than once.
They have 2 mandatory parameters, both strings. The same rules as
standard_entry apply for quoting, comments and whitespaces.
*)
let renamecmd_entry = [ indent . key renamecmd . del_ws_spc . from . del_ws_spc . to . eol ]

let cobl_cmd = /client-output-buffer-limit/
let class = [ label "class" . Quote.do_quote_opt_nil (store Rx.word) ]
let hard_limit = [ label "hard_limit" . Quote.do_quote_opt_nil (store Rx.word) ]
let soft_limit = [ label "soft_limit" . Quote.do_quote_opt_nil (store Rx.word) ]
let soft_seconds = [ label "soft_seconds" . Quote.do_quote_opt_nil (store Rx.integer) ]
(* View: client_output_buffer_limit_entry
Entries identified by the "client-output-buffer-limit" keyword can be found
more than once. They have four mandatory parameters, of which the first is a
string, the last one is an integer and the others are either integers or words,
although redis is very liberal and takes "4242yadayadabytes" as a valid limit.
The same rules as standard_entry apply for quoting, comments and whitespaces.
*)
let client_output_buffer_limit_entry =
  [ indent . key cobl_cmd . del_ws_spc . class . del_ws_spc . hard_limit .
    del_ws_spc . soft_limit . del_ws_spc . soft_seconds . eol ]

let entry = standard_entry
          | save_entry
	  | renamecmd_entry
	  | replicaof_entry
	  | bind_entry
    | sentinel_entry
	  | client_output_buffer_limit_entry

(* View: lns
The Redis lens
*)
let lns = (comment | empty | entry )*

let filter =
    incl "/etc/redis.conf"
  . incl "/etc/redis/redis.conf"

let xfm = transform lns filter
