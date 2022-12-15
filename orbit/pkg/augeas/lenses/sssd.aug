(* 
Module Sssd  
  Lens for parsing sssd.conf

Author: Erinn Looney-Triggs <erinn.looneytriggs@gmail.com>

About: License
  This file is licensed under the LGPL v2+, like the rest of Augeas.

About: Configuration files
   This lens applies to /etc/sssd/sssd.conf. See <filter>.
*)

module Sssd =
  autoload xfm

let comment  = IniFile.comment /[#;]/ "#"

let sep      = IniFile.sep "=" "="

let entry    = IniFile.indented_entry IniFile.entry_re sep comment

(* View: title
    An sssd.conf section title *)
let title   = IniFile.indented_title_label "target" IniFile.record_label_re

(* View: record
    An sssd.conf record *)
let record  = IniFile.record title entry

(* View: lns 
    The sssd.conf lens *)
let lns    = ( comment | IniFile.empty )* . (record)* 

(* View: filter *)
let filter = (incl "/etc/sssd/sssd.conf")

let xfm = transform lns filter

