(* Dnsmasq module for Augeas
 Author: Free Ekanayaka <free@64studio.com>

 Reference: man dnsmasq (8)

 "Format is one option per line, legal options are the same
  as the long options legal on the command line. See
 "/usr/sbin/dnsmasq --help" or "man 8 dnsmasq" for details."

*)

module Dnsmasq =

   autoload xfm

(************************************************************************
 *                           USEFUL PRIMITIVES
 *************************************************************************)

let eol          = Util.eol
let spc          = Util.del_ws_spc
let comment      = Util.comment
let empty        = Util.empty

let sep_eq       = Sep.equal
let sto_to_eol   = store /([^ \t\n].*[^ \t\n]|[^ \t\n])/

let slash        = Util.del_str "/"
let sto_no_slash = store /([^\/ \t\n]+)/
let domains      = slash . [ label "domain" . sto_no_slash . slash ]+

(************************************************************************
 *                            SIMPLE ENTRIES
 *************************************************************************)

let entry_re   = Rx.word - /(address|server)/
let entry      = [ key entry_re . (sep_eq . sto_to_eol)? . eol ]

(************************************************************************
 *                          STRUCTURED ENTRIES
 *************************************************************************)

let address       = [ key "address" . sep_eq . domains . sto_no_slash . eol ]

let server        =
     let port     = [ Build.xchgs "#" "port" . store Rx.integer ]
  in let source   = [ Build.xchgs "@" "source" . store /[^#\/ \t\n]+/ . port? ]
  in let srv_spec = store /(#|([^#@\/ \t\n]+))/ . port? . source?
  in [ key "server" . sep_eq . domains? . srv_spec? . eol ]

(************************************************************************
 *                                LENS
 *************************************************************************)

let lns = (comment|empty|address|server|entry) *

let filter            = incl "/etc/dnsmasq.conf"
                      . incl "/etc/dnsmasq.d/*"
                      . excl ".*"
                      . Util.stdexcl

let xfm                = transform lns filter
