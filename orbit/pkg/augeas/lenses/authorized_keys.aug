(*
Module: Authorized_Keys
  Parses SSH authorized_keys

Author: Raphael Pinson <raphael.pinson@camptocamp.com>

About: Reference
  This lens tries to keep as close as possible to `man 5 authorized_keys` where possible.

About: License
   This file is licenced under the LGPL v2+, like the rest of Augeas.

About: Lens Usage
   To be documented

About: Configuration files
   This lens applies to SSH authorized_keys. See <filter>.

About: Examples
   The <Test_Authorized_Keys> file contains various examples and tests.
*)


module Authorized_Keys =

autoload xfm

(* View: option
   A key option *)
let option =
     let kv_re   = "command" | "environment" | "from"
                 | "permitopen" | "principals" | "tunnel"
  in let flag_re = "cert-authority" | "no-agent-forwarding"
                 | "no-port-forwarding" | "no-pty" | "no-user-rc"
                 | "no-X11-forwarding"
  in let option_value = Util.del_str "\""
                      . store /((\\\\")?[^\\\n"]*)+/
                      . Util.del_str "\""
  in Build.key_value kv_re Sep.equal option_value
   | Build.flag flag_re

(* View: key_options
   A list of key <option>s *)
let key_options = [ label "options" . Build.opt_list option Sep.comma ]

(* View: key_type *)
let key_type =
  let key_type_re = /ecdsa-sha2-nistp[0-9]+/ | /ssh-[a-z0-9]+/
  in [ label "type" . store key_type_re ]

(* View: key_comment *)
let key_comment = [ label "comment" . store Rx.space_in ]

(* View: authorized_key *)
let authorized_key =
   [ label "key"
     . (key_options . Sep.space)?
     . key_type . Sep.space
     . store Rx.no_spaces
     . (Sep.space . key_comment)?
     . Util.eol ]

(* View: lns
     The authorized_keys lens
*)
let lns = ( Util.empty | Util.comment | authorized_key)*

(* Variable: filter *)
let filter = incl (Sys.getenv("HOME") . "/.ssh/authorized_keys")

let xfm = transform lns filter

