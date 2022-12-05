(*
Module: Dhcpd
  BIND dhcp 3 server configuration module for Augeas

Author: Francis Giraldeau <francis.giraldeau@usherbrooke.ca>

About: Reference
  Reference: manual of dhcpd.conf and dhcp-eval
  Follow dhclient module for tree structure

About: License
    This file is licensed under the GPL.

About: Lens Usage
  Sample usage of this lens in augtool

  Directive without argument.
  Set this dhcpd server authoritative on the domain.
  > clear /files/etc/dhcp3/dhcpd.conf/authoritative

  Directives with integer or string argument.
  Set max-lease-time to one hour:
  > set /files/etc/dhcp3/dhcpd.conf/max-lease-time 3600

  Options are declared as a list, even for single values.
  Set the domain of the network:
  > set /files/etc/dhcp3/dhcpd.conf/option/domain-name/arg example.org
  Set two name server:
  > set /files/etc/dhcp3/dhcpd.conf/option/domain-name-servers/arg[1] foo.example.org
  > set /files/etc/dhcp3/dhcpd.conf/option/domain-name-servers/arg[2] bar.example.org

  Create the subnet 172.16.0.1 with 10 addresses:
  > clear /files/etc/dhcp3/dhcpd.conf/subnet[last() + 1]
  > set /files/etc/dhcp3/dhcpd.conf/subnet[last()]/network 172.16.0.0
  > set /files/etc/dhcp3/dhcpd.conf/subnet[last()]/netmask 255.255.255.0
  > set /files/etc/dhcp3/dhcpd.conf/subnet[last()]/range/from 172.16.0.10
  > set /files/etc/dhcp3/dhcpd.conf/subnet[last()]/range/to 172.16.0.20

  Create a new group "foo" with one static host. Nodes type and address are ordered.
  > ins group after /files/etc/dhcp3/dhcpd.conf/subnet[network='172.16.0.0']/*[last()]
  > set /files/etc/dhcp3/dhcpd.conf/subnet[network='172.16.0.0']/group[last()]/host foo
  > set /files/etc/dhcp3/dhcpd.conf/subnet[network='172.16.0.0']/group[host='foo']/host/hardware/type "ethernet"
  > set /files/etc/dhcp3/dhcpd.conf/subnet[network='172.16.0.0']/group[host='foo']/host/hardware/address "00:00:00:aa:bb:cc"
  > set /files/etc/dhcp3/dhcpd.conf/subnet[network='172.16.0.0']/group[host='foo']/host/fixed-address 172.16.0.100

About: Configuration files
  This lens applies to /etc/dhcpd3/dhcpd.conf. See <filter>.
*)

module Dhcpd =

autoload xfm

(************************************************************************
 *                           USEFUL PRIMITIVES
 *************************************************************************)
let dels (s:string)   = del s s
let eol               = Util.eol
let comment           = Util.comment
let empty             = Util.empty
let indent            = Util.indent
let eos               = comment?

(* Define separators *)
let sep_spc           = del /[ \t]+/ " "
let sep_osp           = del /[ \t]*/ ""
let sep_scl           = del /[ \t]*;([ \t]*\n)*/ ";\n"
let sep_obr           = del /[ \t\n]*\{([ \t]*\n)*/ " {\n"
let sep_cbr           = del /[ \t]*\}([ \t]*\n)*/ "}\n"
let sep_com           = del /[ \t\n]*,[ \t\n]*/ ", "
let sep_slh           = del "\/" "/"
let sep_col           = del ":" ":"
let sep_eq            = del /[ \t\n]*=[ \t\n]*/ "="
let scl               = del ";" ";"

(* Define basic types *)
let word              = /[A-Za-z0-9_.-]+(\[[0-9]+\])?/
let ip                = Rx.ipv4

(* Define fields *)

(* adapted from sysconfig.aug *)
  (* Chars allowed in a bare string *)
  let bchar = /[^ \t\n"'\\{}#,()\/]|\\\\./
  let qchar = /["']/  (* " *)

  (* We split the handling of right hand sides into a few cases:
   *   bare  - strings that contain no spaces, optionally enclosed in
   *           single or double quotes
   *   dquot - strings that contain at least one space, apostrophe or slash
   *           which must be enclosed in double quotes
   *   squot - strings that contain an unescaped double quote
   *)
  let bare = del qchar? "" . store (bchar+) . del qchar? ""
  let quote = Quote.do_quote (store (bchar* . /[ \t'\/]/ . bchar*)+)
  let dquote = Quote.do_dquote (store (bchar+))
  (* these two are for special cases.  bare_to_scl is for any bareword that is
   * space or semicolon terminated.  dquote_any allows almost any character in
   * between the quotes. *)
  let bare_to_scl = Quote.do_dquote_opt (store /[^" \t\n;]+/)
  let dquote_any = Quote.do_dquote (store /[^"\n]*[ \t]+[^"\n]*/)

let sto_to_spc        = store /[^\\#,;\{\}" \t\n]+|"[^\\#"\n]+"/
let sto_to_scl        = store /[^ \t;][^;\n=]+[^ \t;]|[^ \t;=]+/

let sto_number        = store /[0-9][0-9]*/

(************************************************************************
 *                         NO ARG STATEMENTS
 *************************************************************************)

let stmt_noarg_re     =   "authoritative"
                        | "primary"
                        | "secondary"

let stmt_noarg        = [ indent
                        . key stmt_noarg_re
                        . sep_scl
                        . eos ]

(************************************************************************
 *                         INT ARG STATEMENTS
 *************************************************************************)

let stmt_integer_re   = "default-lease-time"
                      | "max-lease-time"
                      | "min-lease-time"
                      | /lease[ ]+limit/
                      | "port"
                      | /peer[ ]+port/
                      | "max-response-delay"
                      | "max-unacked-updates"
                      | "mclt"
                      | "split"
                      | /load[ ]+balance[ ]+max[ ]+seconds/
                      | "max-lease-misbalance"
                      | "max-lease-ownership"
                      | "min-balance"
                      | "max-balance"
                      | "adaptive-lease-time-threshold"
                      | "dynamic-bootp-lease-length"
                      | "local-port"
                      | "min-sec"
                      | "omapi-port"
                      | "ping-timeout"
                      | "remote-port"

let stmt_integer      = [ indent
                        . key stmt_integer_re
                        . sep_spc
                        . sto_number
                        . sep_scl
                        . eos ]

(************************************************************************
 *                         STRING ARG STATEMENTS
 *************************************************************************)

let stmt_string_re    = "ddns-update-style"
                      | "ddns-updates"
                      | "ddns-hostname"
                      | "ddns-domainname"
                      | "ddns-rev-domainname"
                      | "log-facility"
                      | "server-name"
                      | "fixed-address"
                      | /failover[ ]+peer/
                      | "use-host-decl-names"
                      | "next-server"
                      | "address"
                      | /peer[ ]+address/
                      | "type"
                      | "file"
                      | "algorithm"
                      | "secret"
                      | "key"
                      | "include"
                      | "hba"
                      | "boot-unknown-clients"
                      | "db-time-format"
                      | "do-forward-updates"
                      | "dynamic-bootp-lease-cutoff"
                      | "get-lease-hostnames"
                      | "infinite-is-reserved"
                      | "lease-file-name"
                      | "local-address"
                      | "one-lease-per-client"
                      | "pid-file-name"
                      | "ping-check"
                      | "server-identifier"
                      | "site-option-space"
                      | "stash-agent-options"
                      | "update-conflict-detection"
                      | "update-optimization"
                      | "update-static-leases"
                      | "use-host-decl-names"
                      | "use-lease-addr-for-default-route"
                      | "vendor-option-space"
                      | "primary"
                      | "omapi-key"

let stmt_string_tpl (kw:regexp) (l:lens) = [ indent
                        . key kw
                        . sep_spc
                        . l
                        . sep_scl
                        . eos ]

let stmt_string  = stmt_string_tpl stmt_string_re bare
                 | stmt_string_tpl stmt_string_re quote
                 | stmt_string_tpl "filename" dquote

(************************************************************************
 *                         RANGE STATEMENTS
 *************************************************************************)

let stmt_range        = [ indent
                        . key "range"
                        . sep_spc
                        . [ label "flag" . store /dynamic-bootp/ . sep_spc ]?
                        . [ label "from" . store ip . sep_spc ]?
                        . [ label "to" . store ip ]
                        . sep_scl
                        . eos ]

(************************************************************************
 *                         HARDWARE STATEMENTS
 *************************************************************************)

let stmt_hardware     = [ indent
                        . key "hardware"
                        . sep_spc
                        . [ label "type" . store /ethernet|tokenring|fddi/ ]
                        . sep_spc
                        . [ label "address" . store /[a-fA-F0-9:-]+/ ]
                        . sep_scl
                        . eos ]

(************************************************************************
 *                         SET STATEMENTS
 *************************************************************************)
let stmt_set          = [ indent
                        . key "set"
                        . sep_spc
                        . store word
                        . sep_spc
                        . Sep.equal
                        . sep_spc
                        . [ label "value" . sto_to_scl ]
                        . sep_scl
                        . eos ]

(************************************************************************
 *                         OPTION STATEMENTS
 *************************************************************************)
(* The general case is considering options as a list *)


let stmt_option_value = /((array of[ \t]+)?(((un)?signed[ \t]+)?integer (8|16|32)|string|ip6?-address|boolean|domain-list|text)|encapsulate [A-Za-z0-9_.-]+)/

let stmt_option_list  = ([ label "arg" . bare ] | [ label "arg" . quote ])
                        . ( sep_com . ([ label "arg" . bare ] | [ label "arg" . quote ]))*

let del_trail_spc = del /[ \t\n]*/ ""

let stmt_record = counter "record" . Util.del_str "{"
                . sep_spc
                . ([seq "record" . store stmt_option_value . sep_com]*
                .  [seq "record" . store stmt_option_value . del_trail_spc])?
                . Util.del_str "}"

let stmt_option_code  = [ label "label" . store word . sep_spc ]
                        . [ key "code" . sep_spc . store word ]
                        . sep_eq
                        . ([ label "type" . store stmt_option_value ]
                          |[ label "record" . stmt_record ]) 

let stmt_option_basic = [ key word . sep_spc . stmt_option_list ]
let stmt_option_extra = [ key word . sep_spc . store /true|false/ . sep_spc . stmt_option_list ]

let stmt_option_body = stmt_option_basic | stmt_option_extra

let stmt_option1  = [ indent
                        . key "option"
                        . sep_spc
                        . stmt_option_body
                        . sep_scl
                        . eos ]

let stmt_option2  = [ indent
                        . dels "option" . label "rfc-code"
                        . sep_spc
                        . stmt_option_code
                        . sep_scl
                        . eos ]

let stmt_option = stmt_option1 | stmt_option2

(************************************************************************
 *                         SUBCLASS STATEMENTS
 *************************************************************************)
(* this statement is not well documented in the manual dhcpd.conf
   we support basic use case *)

let stmt_subclass = [ indent . key "subclass" . sep_spc 
                      . ( [ label "name" .  bare_to_scl ]|[ label "name" .  dquote_any ] )
                      . sep_spc 
                      . ( [ label "value" . bare_to_scl ]|[ label "value" . dquote_any ] ) 
                      . sep_scl 
                      . eos ]


(************************************************************************
 *                         ALLOW/DENY STATEMENTS
 *************************************************************************)
(* We have to use special key for allow/deny members of
  to avoid ambiguity in the put direction *)

let allow_deny_re     = /unknown(-|[ ]+)clients/
                      | /known(-|[ ]+)clients/
                      | /all[ ]+clients/
                      | /dynamic[ ]+bootp[ ]+clients/
                      | /authenticated[ ]+clients/
                      | /unauthenticated[ ]+clients/
                      | "bootp"
                      | "booting"
                      | "duplicates"
                      | "declines"
                      | "client-updates"
                      | "leasequery"

let stmt_secu_re      = "allow"
                      | "deny"

let del_allow = del /allow[ ]+members[ ]+of/ "allow members of"
let del_deny  = del /deny[ \t]+members[ \t]+of/ "deny members of"

(* bare is anything but whitespace, quote marks or semicolon.
 * technically this should be locked down to mostly alphanumerics, but the
 * idea right now is just to make things work.  Also ideally I would use
 * dquote_space but I had a whale of a time with it.  It doesn't like
 * semicolon termination and my attempts to fix that led me to 3 hours of
 * frustration and back to this :)
 *)
let stmt_secu_tpl (l:lens) (s:string) =
                  [ indent . l . sep_spc . label s . bare_to_scl . sep_scl . eos ] |
                  [ indent . l . sep_spc . label s . dquote_any . sep_scl . eos ]


let stmt_secu         = [ indent . key stmt_secu_re . sep_spc .
                          store allow_deny_re . sep_scl . eos ] |
                        stmt_secu_tpl del_allow "allow-members-of" |
                        stmt_secu_tpl del_deny "deny-members-of"

(************************************************************************
 *                         MATCH STATEMENTS
 *************************************************************************)

let sto_com = /[^ \t\n,\(\)][^,\(\)]*[^ \t\n,\(\)]|[^ \t\n,\(\)]+/ | word . /[ \t]*\([^)]*\)/
(* this is already the most complicated part of this module and it's about to
 * get worse.  match statements can be way more complicated than this
 *
 * examples:
 *      using or:
 *      match if ((option vendor-class-identifier="Banana Bready") or (option vendor-class-identifier="Cherry Sunfire"));
 *      unneeded parenthesis:
 *      match if (option vendor-class-identifier="Hello");
 *
 *      and of course the fact that the above two rules used one of infinately
 *      many potential options instead of a builtin function.
 *)
(* sto_com doesn't support quoted strings as arguments.  It also doesn't
   support single arguments (needs to match a comma) It will need to be
   updated for lcase, ucase and log to be workable.

   it also doesn't support no arguments, so gethostbyname() doesn't work.

   option and config-option are considered operators.  They should be matched
   in stmt_entry but also available under "match if" and "if" conditionals
   leased-address, host-decl-name, both take no args and return a value.  We
   might need to treat them as variable names in the parser.

   things like this may be near-impossible to parse even with recursion
   because we have no way of knowing when or if a subfunction takes arguments
   set ClientMac = binary-to-ascii(16, 8, ":", substring(hardware, 1, 6));

   even if we could parse it, they could get arbitrarily complicated like:
   binary-to-ascii(16, 8, ":", substring(hardware, 1, 6) and substring(hardware, 2, 3));

   so at some point we may need to programmatically knock it off and tell
   people to put weird stuff in an include file that augeas doesn't parse.

   the other option is to change the API to not parse the if statement at all,
   just pull in the conditional as a string.
 *)

let fct_re = "substring" | "binary-to-ascii" | "suffix" | "lcase" | "ucase"
             | "gethostbyname" | "packet"
             | "concat" | "reverse" | "encode-int"
             | "extract-int" | "lease-time" | "client-state" | "exists" | "known" | "static"
             | "pick-first-value" | "log" | "execute"

(* not needs to be different because it's a negation of whatever happens next *)
let op_re = "~="|"="|"~~"|"and"|"or"

let fct_args = [ label "args" . dels "(" . sep_osp .
                 ([ label "arg" . store sto_com ] . [ label "arg" . sep_com . store sto_com ]+) .
                        sep_osp . dels ")" ]

let stmt_match_ifopt = [ dels "if" . sep_spc . key "option" . sep_spc . store word .
                      sep_eq . ([ label "value" . bare_to_scl ]|[ label "value" . dquote_any ]) ]

let stmt_match_func = [ store fct_re . sep_osp . label "function" . fct_args ] .
                      sep_eq . ([ label "value" . bare_to_scl ]|[ label "value" . dquote_any ])

let stmt_match_pfv = [ label "function" . store "pick-first-value" . sep_spc .
                       dels "(" . sep_osp .
                       [ label "args" .
                         [ label "arg" . store sto_com ] .
                         [ sep_com . label "arg" . store sto_com ]+ ] .
                       dels ")" ]

let stmt_match_tpl (l:lens) = [ indent . key "match" . sep_spc . l . sep_scl . eos ]

let stmt_match = stmt_match_tpl (dels "if" . sep_spc . stmt_match_func | stmt_match_pfv | stmt_match_ifopt)

(************************************************************************
 *                         BLOCK STATEMENTS
 *************************************************************************)
(* Blocks doesn't support comments at the end of the closing bracket *)

let stmt_entry        =   stmt_secu
                        | stmt_option
                        | stmt_hardware
                        | stmt_range
                        | stmt_string
                        | stmt_integer
                        | stmt_noarg
                        | stmt_match
                        | stmt_subclass
                        | stmt_set
                        | empty
                        | comment

let stmt_block_noarg_re = "pool" | "group"

let stmt_block_noarg (body:lens)
                        = [ indent
                        . key stmt_block_noarg_re
                        . sep_obr
                        . body*
                        . sep_cbr ]

let stmt_block_arg_re = "host"
                      | "class"
                      | "shared-network"
                      | /failover[ ]+peer/
                      | "zone"
                      | "group"
                      | "on"

let stmt_block_arg (body:lens)
                      = ([ indent . key stmt_block_arg_re . sep_spc . dquote_any . sep_obr . body* . sep_cbr ]
                         |[ indent . key stmt_block_arg_re . sep_spc . bare_to_scl . sep_obr . body* . sep_cbr ]
                         |[ indent . del /key/ "key" . label "key_block" . sep_spc . dquote_any . sep_obr . body* . sep_cbr . del /(;([ \t]*\n)*)?/ ""  ]
                         |[ indent . del /key/ "key" . label "key_block" . sep_spc . bare_to_scl . sep_obr . body* . sep_cbr . del /(;([ \t]*\n)*)?/ "" ])

let stmt_block_subnet (body:lens)
                      = [ indent
                        . key "subnet"
                        . sep_spc
                        . [ label "network" . store ip ]
                        . sep_spc
                        . [ key "netmask" . sep_spc . store ip ]
                        . sep_obr
                        . body*
                        . sep_cbr ]

let conditional (body:lens) =
     let condition         = /[^{ \r\t\n][^{\n]*[^{ \r\t\n]|[^{ \t\n\r]/
  in let elsif = [ indent
                 . Build.xchgs "elsif" "@elsif"
                 . sep_spc
                 . store condition
                 . sep_obr
                 . body*
                 . sep_cbr ]
  in let else = [  indent
                 . Build.xchgs "else" "@else"
                 . sep_obr
                 . body*
                 . sep_cbr ]
  in [ indent
     . Build.xchgs "if" "@if"
     . sep_spc
     . store condition
     . sep_obr
     . body*
     . sep_cbr
     . elsif*
     . else? ]


let all_block (body:lens) =
    let lns1 = stmt_block_subnet body in
    let lns2 = stmt_block_arg body in
    let lns3 = stmt_block_noarg body in
    let lns4 = conditional body in
    (lns1 | lns2 | lns3 | lns4 | stmt_entry)

let rec lns_staging = stmt_entry|all_block lns_staging
let lns = (lns_staging)*

let filter = incl "/etc/dhcp3/dhcpd.conf"
           . incl "/etc/dhcp/dhcpd.conf"
           . incl "/etc/dhcpd.conf"

let xfm = transform lns filter
