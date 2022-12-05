(*
Module: Collectd
  Parses collectd configuration files

Author: Raphael Pinson <raphink@gmail.com>

About: Reference
  This lens tries to keep as close as possible to `man 5 collectd.conf` where possible.

About: License
   This file is licenced under the LGPL v2+, like the rest of Augeas.

About: Lens Usage
   To be documented

About: Configuration files
   This lens applies to collectd configuration files. See <filter>.

About: Examples
   The <Test_Collectd> file contains various examples and tests.
*)

module Collectd =

autoload xfm

(* View: lns
    Collectd is essentially Httpd-compliant configuration files *)
let lns = Httpd.lns

(* Variable: filter *)
let filter = incl "/etc/collectd.conf"
           . incl "/etc/collectd/*.conf"
           . incl "/usr/share/doc/collectd/examples/collection3/etc/collection.conf"

let xfm = transform lns filter
