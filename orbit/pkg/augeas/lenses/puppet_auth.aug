(*
Module: Puppet_Auth
  Parses /etc/puppet/auth.conf

Author: Raphael Pinson <raphael.pinson@camptocamp.com>

About: Reference
  This lens tries to keep as close as possible to `http://docs.puppetlabs.com/guides/rest_auth_conf.html` where possible.

About: License
   This file is licenced under the LGPL v2+, like the rest of Augeas.

About: Lens Usage
   To be documented

About: Configuration files
   This lens applies to /etc/puppet/auth.conf. See <filter>.

About: Examples
   The <Test_Puppet_Auth> file contains various examples and tests.
*)

module Puppet_Auth =

autoload xfm

(* View: list
   A list of values *)
let list (kw:string) (val:regexp) =
     let item = [ seq kw . store val ]
  in let comma = del /[ \t]*,[ \t]*/ ", "
  in [ Util.indent . key kw . Sep.space
     . Build.opt_list item comma . Util.comment_or_eol ]

(* View: auth
   An authentication stanza *)
let auth =
  [ Util.indent . Build.xchg /auth(enticated)?/ "auth" "auth"
  . Sep.space . store /yes|no|on|off|any/ . Util.comment_or_eol ]

(* View: reset_counters *)
let reset_counters =
    counter "environment" . counter "method"
  . counter "allow" . counter "allow_ip"

(* View: setting *)
let setting = list "environment" Rx.word
            | list "method" /find|search|save|destroy/
            | list "allow" /[^# \t\n,][^#\n,]*[^# \t\n,]|[^# \t\n,]/
            | list "allow_ip" /[A-Za-z0-9.:\/]+/
            | auth

(* View: record *)
let record =
     let operator = [ label "operator" . store "~" ]
  in [ Util.indent . key "path"
      . (Sep.space . operator)?
      . Sep.space . store /[^~# \t\n][^#\n]*[^# \t\n]|[^~# \t\n]/ . Util.eol
      . reset_counters
      . (Util.empty | Util.comment | setting)*
      . setting ]

(* View: lns *)
let lns = (Util.empty | Util.comment | record)*

(* Variable: filter *)
let filter = (incl "/etc/puppet/auth.conf"
             .incl "/usr/local/etc/puppet/auth.conf"
             .incl "/etc/puppetlabs/puppet/auth.conf")

let xfm = transform lns filter
