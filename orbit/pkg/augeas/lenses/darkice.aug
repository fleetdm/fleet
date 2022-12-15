(* Darkice module for Augeas
   Author: Free Ekanayaka <free@64studio.com>

   Reference: man 5 darkice.cfg
*)


module Darkice =
  autoload xfm


(************************************************************************
 * INI File settings
 *************************************************************************)
let comment  = IniFile.comment IniFile.comment_re IniFile.comment_default

let sep      = IniFile.sep IniFile.sep_re IniFile.sep_default

let entry_re = ( /[A-Za-z0-9][A-Za-z0-9._-]*/ )
let entry = IniFile.entry entry_re sep comment

let title   = IniFile.title_label "target" IniFile.record_label_re
let record  = IniFile.record title entry

let lns    = IniFile.lns record comment

let filter = (incl "/etc/darkice.cfg")

let xfm = transform lns filter
