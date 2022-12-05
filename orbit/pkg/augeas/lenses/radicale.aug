(* Radicale module for Augeas
 Based on Puppet lens.

 Manage config file for http://radicale.org/
 /etc/radicale/config is a standard INI File.
*)


module Radicale =
  autoload xfm

(************************************************************************
 * INI File settings
 *
 * /etc/radicale/config only supports "#" as commentary and "=" as separator
 *************************************************************************)
let comment    = IniFile.comment "#" "#"
let sep        = IniFile.sep "=" "="


(************************************************************************
 *                        ENTRY
 * /etc/radicale/config uses standard INI File entries
 *************************************************************************)
let entry   = IniFile.indented_entry IniFile.entry_re sep comment


(************************************************************************
 *                        RECORD
 * /etc/radicale/config uses standard INI File records
 *************************************************************************)
let title   = IniFile.indented_title IniFile.record_re
let record  = IniFile.record title entry


(************************************************************************
 *                        LENS & FILTER
 * /etc/radicale/config uses standard INI File records
 *************************************************************************)
let lns     = IniFile.lns record comment

let filter = (incl "/etc/radicale/config")

let xfm = transform lns filter
