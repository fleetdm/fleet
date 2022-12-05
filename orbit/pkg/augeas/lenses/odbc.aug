(*
 ODBC lens for Augeas
 Author: Marc Fournier <marc.fournier@camptocamp.com>

 odbc.ini and odbcinst.ini are standard INI files.
*)


module Odbc =
  autoload xfm

(************************************************************************
 *                     INI File settings
 * odbc.ini only supports "# as commentary and "=" as separator
 ************************************************************************)
let comment    = IniFile.comment "#" "#"
let sep        = IniFile.sep "=" "="


(************************************************************************
 *                        ENTRY
 * odbc.ini uses standard INI File entries
 ************************************************************************)
let entry   = IniFile.indented_entry IniFile.entry_re sep comment


(************************************************************************
 *                        RECORD
 * odbc.ini uses standard INI File records
 ************************************************************************)
let title   = IniFile.indented_title IniFile.record_re
let record  = IniFile.record title entry


(************************************************************************
 *                        LENS & FILTER
 ************************************************************************)
let lns     = IniFile.lns record comment

let filter = incl "/etc/odbc.ini"
           . incl "/etc/odbcinst.ini"

let xfm = transform lns filter
