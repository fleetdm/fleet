(*
Module: Keepalived
  Parses /etc/ld.so.conf and /etc/ld.so.conf.d/*

Author: Raphael Pinson <raphink@gmail.com>

About: License
   This file is licenced under the LGPL v2+, like the rest of Augeas.

About: Lens Usage
   To be documented

About: Configuration files
   This lens applies to /etc/ld.so.conf and /etc/ld.so.conf.d/*. See <filter>.

About: Examples
   The <Test_Ldso> file contains various examples and tests.
*)

module LdSo =

autoload xfm

(* View: path *)
let path = [ label "path" . store /[^# \t\n][^ \t\n]*/ . Util.eol ]

(* View: include *)
let include = Build.key_value_line "include" Sep.space (store Rx.fspath)

(* View: hwcap *)
let hwcap =
    let hwcap_val = [ label "bit" . store Rx.integer ] . Sep.space .
                      [ label "name" . store Rx.word ]
  in Build.key_value_line "hwcap" Sep.space hwcap_val

(* View: lns *)
let lns = (Util.empty | Util.comment | path | include | hwcap)*

(* Variable: filter *)
let filter = incl "/etc/ld.so.conf"
           . incl "/etc/ld.so.conf.d/*"
           . Util.stdexcl

let xfm = transform lns filter
