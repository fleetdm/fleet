(* Parsing yum's config files *)
module Yum =
  autoload xfm

(************************************************************************
 * INI File settings
 *************************************************************************)

let comment  = IniFile.comment "#" "#"
let sep      = IniFile.sep "=" "="
let empty    = Util.empty
let eol      = IniFile.eol

(************************************************************************
 *                        ENTRY
 *************************************************************************)

let list_entry (list_key:string)  =
  let list_value = store /[^# \t\r\n,][^ \t\r\n,]*[^# \t\r\n,]|[^# \t\r\n,]/ in
  let list_sep = del /([ \t]*(,[ \t]*|\r?\n[ \t]+))|[ \t]+/ "\n\t" in
  [ key list_key . sep . Sep.opt_space . list_value ]
  . (list_sep . Build.opt_list [ label list_key . list_value ] list_sep)?
  . eol

let entry_re = IniFile.entry_re - ("baseurl" | "gpgkey" | "exclude")

let entry       = IniFile.entry entry_re sep comment
                | empty

let entries =
     let list_entry_elem (k:string) = list_entry k . entry*
  in entry*
   | entry* . Build.combine_three_opt
                (list_entry_elem "baseurl")
                (list_entry_elem "gpgkey")
                (list_entry_elem "exclude")


(***********************************************************************a
 *                         TITLE
 *************************************************************************)
let title       = IniFile.title IniFile.record_re
let record      = [ title . entries ]


(************************************************************************
 *                         LENS & FILTER
 *************************************************************************)
let lns    = (empty | comment)* . record*

  let filter = (incl "/etc/yum.conf")
      . (incl "/etc/yum.repos.d/*.repo")
      . (incl "/etc/yum/yum-cron*.conf") 
      . (incl "/etc/yum/pluginconf.d/*")
      . (excl "/etc/yum/pluginconf.d/versionlock.list")
      . (incl "/etc/dnf/dnf.conf")
      . (incl "/etc/dnf/automatic.conf")
      . (incl "/etc/dnf/plugins/*.conf")
      . Util.stdexcl

  let xfm = transform lns filter

(* Local Variables: *)
(* mode: caml       *)
(* End:             *)
