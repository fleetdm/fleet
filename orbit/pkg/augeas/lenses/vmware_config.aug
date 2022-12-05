(*
Module: VWware_Config
  Parses /etc/vmware/config

Author: Raphael Pinson <raphael.pinson@camptocamp.com>

About: License
   This file is licenced under the LGPL v2+, like the rest of Augeas.

About: Lens Usage
   To be documented

About: Configuration files
   This lens applies to /etc/vmware/config. See <filter>.

About: Examples
   The <Test_VMware_Config> file contains various examples and tests.
*)
module VMware_Config =

autoload xfm

(* View: entry *)
let entry =
  Build.key_value_line Rx.word Sep.space_equal Quote.double_opt

(* View: lns *)
let lns = (Util.empty | Util.comment | entry)*


(* Variable: filter *)
let filter = incl "/etc/vmware/config"

let xfm = transform lns filter
