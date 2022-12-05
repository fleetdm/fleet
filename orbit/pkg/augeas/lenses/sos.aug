(*
Module: Sos
    Parses Anaconda's user interaction configuration files.

Author: George Hansper <george@hansper.id.au>

About: Reference
    https://github.com/hercules-team/augeas/wiki/Generic-modules-IniFile
    https://github.com/sosreport/sos

About: Configuration file
    This lens applies to /etc/sos/sos.conf

About: License
  This file is licensed under the LGPL v2+, like the rest of Augeas.
*)
module Sos =
autoload xfm

let comment = IniFile.comment "#" "#"
let sep     = IniFile.sep "=" "="

let entry   = IniFile.entry IniFile.entry_re sep comment
let title   = IniFile.title IniFile.record_re
let record  = IniFile.record title entry

let lns     = IniFile.lns record comment

let filter  = ( incl "/etc/sos/sos.conf" )
              . ( Util.stdexcl )

let xfm     = transform lns filter
