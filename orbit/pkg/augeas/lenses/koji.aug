(*
Module: Koji
  Parses koji config files

Author: Pat Riehecky <riehecky@fnal.gov>

About: Reference
  This lens tries to keep as close as possible to koji config syntax

About: License
  This file is licenced under the LGPL v2+, like the rest of Augeas.

About: Lens Usage
  To be documented

About: Configuration files
  This lens applies to:
    /etc/koji.conf
    /etc/kojid/kojid.conf
    /etc/koji-hub/hub.conf
    /etc/kojira/kojira.conf
    /etc/kojiweb/web.conf
    /etc/koji-shadow/koji-shadow.conf 

  See <filter>.
*)

module Koji =
  autoload xfm

let lns     = IniFile.lns_loose_multiline

let filter = incl "/etc/koji.conf"
           . incl "/etc/kojid/kojid.conf"
           . incl "/etc/koji-hub/hub.conf"
           . incl "/etc/kojira/kojira.conf"
           . incl "/etc/kojiweb/web.conf"
           . incl "/etc/koji-shadow/koji-shadow.conf"

let xfm = transform lns filter
