(*
Module: Splunk
  Parses /opt/splunk/etc/*, /opt/splunkforwarder/etc/system/local/*.conf and /opt/splunkforwarder/etc/apps/*/(default|local)/*.conf

Author: Tim Brigham, Jason Antman

About: Reference
  https://docs.splunk.com/Documentation/Splunk/6.5.0/Admin/AboutConfigurationFiles

About: License
   This file is licenced under the LGPL v2+

About: Lens Usage
   Works like IniFile lens, with anonymous section for entries without enclosing section and allowing underscore-prefixed keys.

About: Configuration files
   This lens applies to conf files under /opt/splunk/etc and /opt/splunkforwarder/etc See <filter>.

About: Examples
   The <Test_Splunk> file contains various examples and tests.
*)

module Splunk =
  autoload xfm

  let comment   = IniFile.comment IniFile.comment_re IniFile.comment_default
  let sep       = IniFile.sep IniFile.sep_re IniFile.sep_default
  let empty     = IniFile.empty

  let entry_re  = ( /[A-Za-z_][A-Za-z0-9._-]*/ )
  let setting   = entry_re
  let title     =  IniFile.indented_title_label "target" IniFile.record_label_re
  let entry     = [ key entry_re . sep . IniFile.sto_to_eol? . IniFile.eol ] | comment


  let record    = IniFile.record title entry
  let anon      = [ label ".anon" . (entry|empty)+ ]
  let lns       = anon . (record)* | (record)*

  let filter    = incl "/opt/splunk/etc/system/local/*.conf"
                . incl "/opt/splunk/etc/apps/*/local/*.conf"
                . incl "/opt/splunkforwarder/etc/system/local/*.conf"
                . incl "/opt/splunkforwarder/etc/apps/*/default/*.conf"
                . incl "/opt/splunkforwarder/etc/apps/*/local/*.conf"
  let xfm       = transform lns filter
