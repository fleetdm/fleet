(*
Module: Carbon
    Parses Carbon's configuration files

Author: Marc Fournier <marc.fournier@camptocamp.com>

About: Reference
    This lens is based on the conf/*.conf.example files from the Carbon
    package.

About: Configuration files
    This lens applies to most files in /etc/carbon/. See <filter>.
    NB: whitelist.conf and blacklist.conf use a different syntax. This lens
    doesn't support them.

About: Usage Example
(start code)
    $ augtool
    augtool> ls /files/etc/carbon/carbon.conf/
    cache/ = (none)
    relay/ = (none)
    aggregator/ = (none)

    augtool> get /files/etc/carbon/carbon.conf/cache/ENABLE_UDP_LISTENER
    /files/etc/carbon/carbon.conf/cache/ENABLE_UDP_LISTENER = False

    augtool> set /files/etc/carbon/carbon.conf/cache/ENABLE_UDP_LISTENER True
    augtool> save
    Saved 1 file(s)
(end code)
   The <Test_Carbon> file also contains various examples.

About: License
  This file is licensed under the LGPL v2+, like the rest of Augeas.
*)
module Carbon =
autoload xfm

let comment = IniFile.comment "#" "#"
let sep     = IniFile.sep "=" "="

let entry   = IniFile.entry IniFile.entry_re sep comment
let title   = IniFile.title IniFile.record_re
let record  = IniFile.record title entry

let lns     = IniFile.lns record comment

let filter  = incl "/etc/carbon/carbon.conf"
            . incl "/etc/carbon/relay-rules.conf"
            . incl "/etc/carbon/rewrite-rules.conf"
            . incl "/etc/carbon/storage-aggregation.conf"
            . incl "/etc/carbon/storage-schemas.conf"

let xfm     = transform lns filter
