(*
Module: Dns_Zone
  Lens for parsing DNS zone files

Authors:
  Kaarle Ritvanen <kaarle.ritvanen@datakunkku.fi>

About: Reference
  RFC 1035, RFC 2782, RFC 3403

About: License
  This file is licensed under the LGPL v2+
*)

module Dns_Zone =

autoload xfm

let eol = del /([ \t\n]*(;[^\n]*)?\n)+/ "\n"
let opt_eol = del /([ \t\n]*(;[^\n]*)?\n)*/ ""

let ws = del /[ \t]+|(([ \t\n]*;[^\n]*)?\n)+[ \t]*/ " "
let opt_ws = del /(([ \t\n]*;[^\n]*)?\n)*[ \t]*/ ""

let token = /([^ \t\n";()\\]|\\\\.)+|"([^"\\]|\\\\.)*"/


let control = [ key /\$[^ \t\n\/]+/
                . Util.del_ws_tab
                . store token
                . eol ]


let labeled_token (lbl:string) (re:regexp) (sep:lens) =
    [ label lbl . store re . sep ]

let regexp_token (lbl:string) (re:regexp) =
    labeled_token lbl re Util.del_ws_tab

let type_token (re:regexp) = regexp_token "type" re

let simple_token (lbl:string) = regexp_token lbl token

let enclosed_token (lbl:string) = labeled_token lbl token ws

let last_token (lbl:string) = labeled_token lbl token eol


let class_re = /IN/

let ttl = regexp_token "ttl" /[0-9]+[DHMWdhmw]?/
let class = regexp_token "class" class_re

let rr =
     let simple_type = /[A-Z]+/ - class_re - /MX|NAPTR|SOA|SRV/
  in type_token simple_type . last_token "rdata"


let mx = type_token "MX"
         . simple_token "priority"
         . last_token "exchange"

let naptr = type_token "NAPTR"
            . simple_token "order"
            . simple_token "preference"
            . simple_token "flags"
            . simple_token "service"
            . simple_token "regexp"
            . last_token "replacement"

let soa = type_token "SOA"
          . simple_token "mname"
          . simple_token "rname"
          . Util.del_str "("
          . opt_ws
          . enclosed_token "serial"
          . enclosed_token "refresh"
          . enclosed_token "retry"
          . enclosed_token "expiry"
          . labeled_token "minimum" token opt_ws
          . Util.del_str ")"
          . eol

let srv = type_token "SRV"
         . simple_token "priority"
         . simple_token "weight"
         . simple_token "port"
         . last_token "target"


let record = seq "owner"
             . ((ttl? . class?) | (class . ttl))
             . (rr|mx|naptr|soa|srv)
let ws_record = [ Util.del_ws_tab . record ]
let records (k:regexp) = [ key k . counter "owner" . ws_record+ ]

let any_record_block = records /[^ \t\n;\/$][^ \t\n;\/]*/
let non_root_records = records /@[^ \t\n;\/]+|[^ \t\n;\/$@][^ \t\n;\/]*/

let root_records = [ del /@?/ "@"
                     . Util.del_ws_tab
                     . label "@"
                     . counter "owner"
                     . [ record ]
                     . ws_record* ]

let lns = opt_eol
          . control*
          . ( (root_records|non_root_records)
              . (control|any_record_block)* )?

let filter = incl "/var/bind/pri/*.zone"
let xfm = transform Dns_Zone.lns filter
