(*
Module: Avahi
 Avahi module for Augeas

 Author: Athir Nuaimi <athir@nuaimi.com>

 avahi-daemon.conf is a standard INI File.
*)

module Avahi =
  autoload xfm

(************************************************************************
 * Group: INI File settings
 * avahi-daemon.conf only supports "# as commentary and "=" as separator
 *************************************************************************)
(* View: comment *)
let comment    = IniFile.comment "#" "#"
(* View: sep *)
let sep        = IniFile.sep "=" "="

(************************************************************************
 * Group: Entry
 *************************************************************************)
(* View: entry *)
let entry   = IniFile.indented_entry IniFile.entry_re sep comment

(************************************************************************
 * Group: Record
 *************************************************************************)
(* View: title *)
let title   = IniFile.indented_title IniFile.record_re
(* View: record *)
let record  = IniFile.record title entry

(************************************************************************
 * Group: Lens and filter
 *************************************************************************)
(* View: lns *)
let lns     = IniFile.lns record comment

(* View: filter *)
let filter = (incl "/etc/avahi/avahi-daemon.conf")

let xfm = transform lns filter
