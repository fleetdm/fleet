(* Ceph module for Augeas
 Author: Pavel Chechetin  <pchechetin@mirantis.com>

 ceph.conf is a standard INI File with whitespaces in the title.
*)


module Ceph =
  autoload xfm

let comment    = IniFile.comment IniFile.comment_re IniFile.comment_default
let sep        = IniFile.sep IniFile.sep_re IniFile.sep_default

let entry_re   = /[A-Za-z0-9_.-][A-Za-z0-9 _.-]*[A-Za-z0-9_.-]/

let entry      = IniFile.indented_entry entry_re sep comment

let title   = IniFile.indented_title IniFile.record_re
let record  = IniFile.record title entry

let lns     = IniFile.lns record comment

let filter = (incl "/etc/ceph/ceph.conf")
           . (incl (Sys.getenv("HOME") . "/.ceph/config"))

let xfm = transform lns filter
