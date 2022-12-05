(*
Module: Apt_Update_Manager
  Parses files in /etc/update-manager

Author: Raphael Pinson <raphael.pinson@camptocamp.com>

About: License
   This file is licenced under the LGPL v2+, like the rest of Augeas.

About: Lens Usage
   To be documented

About: Configuration files
   This lens applies to files in /etc/update-manager. See <filter>.

About: Examples
   The <Test_Apt_Update_Manager> file contains various examples and tests.
*)
module Apt_Update_Manager =

autoload xfm

(* View: comment *)
let comment = IniFile.comment IniFile.comment_re IniFile.comment_default

(* View: sep *)
let sep = IniFile.sep IniFile.sep_re IniFile.sep_default

(* View: title *)
let title = IniFile.title Rx.word

(* View: entry *)
let entry = IniFile.entry Rx.word sep comment

(* View: record *)
let record = IniFile.record title entry

(* View: lns *)
let lns = IniFile.lns record comment

(* Variable: filter *)
let filter = incl "/etc/update-manager/meta-release"
           . incl "/etc/update-manager/release-upgrades"
           . incl "/etc/update-manager/release-upgrades.d/*"
           . Util.stdexcl

let xfm = transform lns filter
