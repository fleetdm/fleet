(*
Module: Tinc
  Parses Tinc VPN configuration files

Author: Thomas Weißschuh <thomas.weissschuh@amadeus.com>

About: License
  This file is licensed under the LGPL v2+, like the rest of Augeas.
*)

module Tinc =

autoload xfm

let no_spaces_no_equals = /[^ \t\r\n=]+/
let assign = del (/[ \t]*[= ][ \t]*/) " = "
let del_str = Util.del_str

let entry = Build.key_value_line /[A-Za-z]+/ assign (store no_spaces_no_equals)

let key_section_start = "-----BEGIN RSA PUBLIC KEY-----\n"
let key_section_end = "\n-----END RSA PUBLIC KEY-----"
              (* the last line does not include a newline *)
let base_64 = /[A-Za-z0-9+\/=\n]+[A-Za-z0-9+\/=]/
let key_section = del_str key_section_start .
                  (label "#key" . store base_64) .
                  del_str key_section_end

(* we only support a single key section *)
let lns = (Util.comment | Util.empty | entry) * . [(key_section . Util.empty *)]?

let filter = incl "/etc/tinc.conf"
           . incl "/etc/tinc/*/tinc.conf"
           . incl "/etc/tinc/hosts/*"
           . incl "/etc/tinc/*/hosts/*"

let xfm = transform lns filter
