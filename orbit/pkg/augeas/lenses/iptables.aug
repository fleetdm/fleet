module Iptables =
  autoload xfm

(*
Module: Iptables
   Parse the iptables file format as produced by iptables-save. The
   resulting tree is fairly simple; in particular a rule is simply
   a long list of options/switches and their values (if any)

   This lens should be considered experimental
*)

let comment = Util.comment
let empty = Util.empty
let eol = Util.eol
let spc = Util.del_ws_spc
let dels = Util.del_str

let chain_name = store /[A-Za-z0-9_-]+/
let chain =
  let policy = [ label "policy" . store /ACCEPT|DROP|REJECT|-/ ] in
  let counters_eol = del /[ \t]*(\[[0-9:]+\])?[ \t]*\n/ "\n" in
    [ label "chain" .
        dels ":" . chain_name . spc . policy . counters_eol ]

let param (long:string) (short:string) =
  [ label long .
      spc . del (/--/ . long | /-/ . short) ("-" . short) . spc .
      store /(![ \t]*)?[^ \t\n!-][^ \t\n]*/ ]

(* A negatable parameter, which can either be FTW
     ! --param arg
   or
     --param ! arg
*)
let neg_param (long:string) (short:string) =
  [ label long .
      [ spc . dels "!" . label "not" ]? .
      spc . del (/--/ . long | /-/ . short) ("-" . short) . spc .
      store /(![ \t]*)?[^ \t\n!-][^ \t\n]*/ ]

let tcp_flags =
  let flags = /SYN|ACK|FIN|RST|URG|PSH|ALL|NONE/ in
  let flag_list (name:string) =
    Build.opt_list [label name . store flags] (dels ",") in
  [ label "tcp-flags" .
      spc . dels "--tcp-flags" .
      spc . flag_list "mask" . spc . flag_list "set" ]

(* misses --set-counters *)
let ipt_match =
  let any_key = /[a-zA-Z-][a-zA-Z0-9-]+/ -
    /protocol|source|destination|jump|goto|in-interface|out-interface|fragment|match|tcp-flags/ in
  let any_val = /([^" \t\n!-][^ \t\n]*)|"([^"\\\n]|\\\\.)*"/ in
  let any_param =
    [ [ spc . dels "!" . label "not" ]? .
      spc . dels "--" . key any_key . (spc . store any_val)? ] in
    (neg_param "protocol" "p"
    |neg_param "source" "s"
    |neg_param "destination" "d"
    |param "jump" "j"
    |param "goto" "g"
    |neg_param "in-interface" "i"
    |neg_param "out-interface" "o"
    |neg_param "fragment" "f"
    |param "match" "m"
    |tcp_flags
    |any_param)*

let chain_action (n:string) (o:string) =
    [ label n .
        del (/--/ . n | o) o .
        spc . chain_name . ipt_match . eol ]

let table_rule = chain_action "append" "-A"
	       | chain_action "insert" "-I"
	       | empty


let table = [ del /\*/ "*" . label "table" . store /[a-z]+/ . eol .
                (chain|comment|table_rule)* .
                dels "COMMIT" . eol ]

let lns = (comment|empty|table)*
let xfm = transform lns (incl "/etc/sysconfig/iptables"
                       . incl "/etc/sysconfig/iptables.save"
                       . incl "/etc/iptables-save")
