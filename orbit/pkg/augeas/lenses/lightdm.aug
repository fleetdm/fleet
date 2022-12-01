(* 
Module: Lightdm
  Lightdm module for Augeas for which parses /etc/lightdm/*.conf files which
  are standard INI file format.

Author: David Salmen <dsalmen@dsalmen.com>

About: License
   This file is licenced under the LGPL v2+, like the rest of Augeas.

About: Lens Usage
   To be documented

About: Configuration files
   This lens applies to /etc/lightdm/*.conf.  See <filter>.

About: Tests
   The tests/test_lightdm.aug file contains unit tests.
*)

module Lightdm =
  autoload xfm

(************************************************************************
 * INI File settings
 *
 * lightdm.conf only supports "# as commentary and "=" as separator
 *************************************************************************)
let comment    = IniFile.comment "#" "#"
let sep        = IniFile.sep "=" "="


(************************************************************************
 *                        ENTRY
 * lightdm.conf uses standard INI File entries
 *************************************************************************)
let entry   = IniFile.indented_entry IniFile.entry_re sep comment


(************************************************************************
 *                        RECORD
 * lightdm.conf uses standard INI File records
 *************************************************************************)
let title   = IniFile.indented_title IniFile.record_re
let record  = IniFile.record title entry


(************************************************************************
 *                        LENS & FILTER
 * lightdm.conf uses standard INI File records
 *************************************************************************)
let lns     = IniFile.lns record comment

let filter = (incl "/etc/lightdm/*.conf")

let xfm = transform lns filter
