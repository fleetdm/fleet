(*
Module: Semanage
   Parses /etc/selinux/semanage.conf

Author:
   Pino Toscano <ptoscano@redhat.com>

About: License
   This file is licenced under the LGPL v2+, like the rest of Augeas.

About: Configuration files
   This lens applies to /etc/selinux/semanage.conf. See <filter>.

About: Examples
   The <Test_Semanage> file contains various examples and tests.
*)

module Semanage =
  autoload xfm

let comment = IniFile.comment "#" "#"
let sep = IniFile.sep "=" "="
let empty = IniFile.empty
let eol = IniFile.eol

let list_keys = "ignoredirs"
let scl = del ";" ";"
let fspath = /[^ \t\n;#]+/ (* Rx.fspath without ; or # *)

let entry = IniFile.entry_list list_keys sep fspath scl comment
          | IniFile.entry (IniFile.entry_re - list_keys) sep comment
          | empty

let title = IniFile.title_label "@group" (IniFile.record_re - /^end$/)
let record = [ title . entry+ . Util.del_str "[end]" . eol ]

let lns = (entry | record)*

(* Variable: filter *)
let filter = incl "/etc/selinux/semanage.conf"

let xfm = transform lns filter
