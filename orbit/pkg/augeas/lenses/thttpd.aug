(*
Module: Thttpd
  Parses Thttpd's configuration files

Author: Marc Fournier <marc.fournier@camptocamp.com>

About: Reference
    This lens is based on Thttpd's default thttpd.conf file.

About: Usage Example
(start code)
    augtool> get /files/etc/thttpd/thttpd.conf/port
    /files/etc/thttpd/thttpd.conf/port = 80

    augtool> set /files/etc/thttpd/thttpd.conf/port 8080
    augtool> save
    Saved 1 file(s)
(end code)
   The <Test_Thttpd> file also contains various examples.

About: License
  This file is licensed under the LGPL v2+, like the rest of Augeas.
*)

module Thttpd =
autoload xfm

let comment     = Util.comment
let comment_eol = Util.comment_generic /[ \t]*[#][ \t]*/ " # "
let empty       = Util.empty
let eol         = Util.del_str "\n"
let bol         = Util.del_opt_ws ""

let kvkey       = /(port|dir|data_dir|user|cgipat|throttles|host|logfile|pidfile|charset|p3p|max_age)/
let flag        = /(no){0,1}(chroot|symlinks|vhost|globalpasswd)/
let val         = /[^\n# \t]*/

let kventry     = key kvkey . Util.del_str "=" . store val
let flagentry   = key flag

let kvline      = [ bol . kventry . (eol|comment_eol) ]
let flagline    = [ bol . flagentry . (eol|comment_eol) ]

let lns         = (kvline|flagline|comment|empty)*

let filter      = incl "/etc/thttpd/thttpd.conf"

let xfm         = transform lns filter
