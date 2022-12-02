(*
Module: Memcached
  Parses Memcached's configuration files

Author: Marc Fournier <marc.fournier@camptocamp.com>

About: Reference
    This lens is based on Memcached's default memcached.conf file.

About: Usage Example
(start code)
    augtool> get /files/etc/memcached.conf/u
    /files/etc/memcached.conf/u = nobody

    augtool> set /files/etc/memcached.conf/m 128
    augtool> save
    Saved 1 file(s)
(end code)
   The <Test_Memcached> file also contains various examples.

About: License
  This file is licensed under the LGPL v2+, like the rest of Augeas.
*)

module Memcached =
autoload xfm

let comment     = Util.comment
let comment_eol = Util.comment_generic /[#][ \t]*/ "# "
let option      = /[a-zA-Z]/
let val         = /[^# \n\t]+/
let empty       = Util.empty
let eol         = Util.del_str "\n"

let entry       = [ Util.del_str "-" . key option
                . ( Util.del_ws_spc . (store val) )?
                . del /[ \t]*/ "" . (eol|comment_eol) ]

let logfile     = Build.key_value_line_comment
                  "logfile" Sep.space (store val) comment

let lns         = ( entry | logfile | comment | empty )*

let filter      = incl "/etc/memcached.conf"
                . incl "/etc/memcachedb.conf"

let xfm         = transform lns filter
