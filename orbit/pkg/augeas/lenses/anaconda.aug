(*
Module: Anaconda
    Parses Anaconda's user interaction configuration files.

Author: Pino Toscano <ptoscano@redhat.com>

About: Reference
    https://anaconda-installer.readthedocs.io/en/latest/user-interaction-config-file-spec.html

About: Configuration file
    This lens applies to /etc/sysconfig/anaconda.

About: License
  This file is licensed under the LGPL v2+, like the rest of Augeas.
*)
module Anaconda =
autoload xfm

let comment = IniFile.comment "#" "#"
let sep     = IniFile.sep "=" "="

let entry   = IniFile.entry IniFile.entry_re sep comment
let title   = IniFile.title IniFile.record_re
let record  = IniFile.record title entry

let lns     = IniFile.lns record comment

let filter  = incl "/etc/sysconfig/anaconda"

let xfm     = transform lns filter
