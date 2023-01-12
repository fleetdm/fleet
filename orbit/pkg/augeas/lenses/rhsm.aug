(*
Module: Rhsm
  Parses subscription-manager config files

Author: Dominic Cleal <dcleal@redhat.com>

About: Reference
  This lens tries to keep as close as possible to rhsm.conf(5) and
  Python's SafeConfigParser.  All settings must be in sections without
  indentation.  Semicolons and hashes are permitted for comments.

About: License
  This file is licenced under the LGPL v2+, like the rest of Augeas.

About: Lens Usage
  To be documented

About: Configuration files
  This lens applies to:
    /etc/rhsm/rhsm.conf

  See <filter>.
*)

module Rhsm =
  autoload xfm

(* Semicolons and hashes are permitted for comments *)
let comment = IniFile.comment IniFile.comment_re "#"
(* Equals and colons are permitted for separators *)
let sep     = IniFile.sep IniFile.sep_re IniFile.sep_default

(* All settings must be in sections without indentation *)
let entry   = IniFile.entry_multiline IniFile.entry_re sep comment
let title   = IniFile.title IniFile.record_re
let record  = IniFile.record title entry

let lns     = IniFile.lns record comment

let filter  = incl "/etc/rhsm/rhsm.conf"

let xfm     = transform lns filter
