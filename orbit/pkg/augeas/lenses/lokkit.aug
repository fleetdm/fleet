module Lokkit =
  autoload xfm

(* Module: Lokkit
   Parse the config file for lokkit from system-config-firewall
*)

let comment = Util.comment
let empty = Util.empty
let eol = Util.eol
let spc = Util.del_ws_spc
let dels = Util.del_str

let eq = del /[ \t=]+/ "="
let token = store /[a-zA-Z0-9][a-zA-Z0-9-]*/

let long_opt (n:regexp) =
  [ dels "--" . key n . eq . token . eol ]

let flag (n:regexp) =
  [ dels "--" . key n . eol ]

let option (l:string) (s:string) =
  del ("--" . l | "-" . s) ("--" . l) . label l . eq

let opt (l:string) (s:string) =
  [ option l s . token . eol ]

(* trust directive
   -t <interface>, --trust=<interface>
*)
let trust =
  [ option "trust" "t" . store Rx.device_name . eol ]

(* port directive
   -p <port>[-<port>]:<protocol>, --port=<port>[-<port>]:<protocol>
*)
let port =
  let portnum = store /[0-9]+/ in
  [ option "port" "p" .
    [ label "start" . portnum ] .
    (dels "-" . [ label "end" . portnum])? .
    dels ":" . [ label "protocol" . token ] . eol ]

(* custom_rules directive
   --custom-rules=[<type>:][<table>:]<filename>
*)
let custom_rules =
  let types = store /ipv4|ipv6/ in
  let tables = store /mangle|nat|filter/ in
  let filename = store /[^ \t\n:=][^ \t\n:]*/ in
  [ dels "--custom-rules" . label "custom-rules" . eq .
      [ label "type" . types . dels ":" ]? .
      [ label "table" . tables . dels ":"]? .
      filename . eol ]

(* forward_port directive
   --forward-port=if=<interface>:port=<port>:proto=<protocol>[:toport=<destination port>][:toaddr=<destination address>]
*)
let forward_port =
  let elem (n:string) (v:lens) =
    [ key n . eq . v ] in
  let ipaddr = store /[0-9.]+/ in
  let colon = dels ":" in
  [ dels "--forward-port" . label "forward-port" . eq .
      elem "if" token . colon .
      elem "port" token . colon .
      elem "proto" token .
      (colon . elem "toport" token)? .
      (colon . elem "toaddr" ipaddr)? . eol ]

let entry =
  long_opt /selinux|selinuxtype|addmodule|removemodule|block-icmp/
 |flag /enabled|disabled/
 |opt "service" "s"
 |port
 |trust
 |opt "masq" "m"
 |custom_rules
 |forward_port

let lns = (comment|empty|entry)*

let xfm = transform lns (incl "/etc/sysconfig/system-config-firewall")
