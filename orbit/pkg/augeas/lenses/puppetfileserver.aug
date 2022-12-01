(* -*- coding: utf-8 -*-
Module: PuppetFileserver
  Parses /etc/puppet/fileserver.conf used by puppetmasterd daemon.

Author: Frédéric Lespez <frederic.lespez@free.fr>

About: Reference
  This lens tries to keep as close as possible to puppet documentation
  for this file:
  http://docs.puppetlabs.com/guides/file_serving.html

About: License
  This file is licensed under the LGPL v2+, like the rest of Augeas.

About: Lens Usage
  Sample usage of this lens in augtool

    * Create a new mount point
      > ins test_mount after /files/etc/puppet/fileserver.conf/*[last()]
      > defvar test_mount /files/etc/puppet/fileserver.conf/test_mount
      > set $test_mount/path /etc/puppet/files
      > set $test_mount/allow *.example.com
      > ins allow after $test_mount/*[last()]
      > set $test_mount/allow[last()] server.domain.com
      > set $test_mount/deny dangerous.server.com
    * List the definition of a mount point
      > print /files/etc/puppet/fileserver.conf/files
    * Remove a mount point
      > rm /files/etc/puppet/fileserver.conf/test_mount

About: Configuration files
  This lens applies to /etc/puppet/fileserver.conf. See <filter>.
*)


module PuppetFileserver =
  autoload xfm

(************************************************************************
 * Group:                 USEFUL PRIMITIVES
 *************************************************************************)

(* Group: INI File settings *)

(* Variable: eol *)
let eol = IniFile.eol

(* Variable: sep
  Only treat one space as the sep, extras are stripped by IniFile *)
let sep = Util.del_str " "

(*
Variable: comment
  Only supports "#" as commentary
*)
let comment = IniFile.comment "#" "#"

(*
Variable: entry_re
  Regexp for possible <entry> keyword (path, allow, deny)
*)
let entry_re = /path|allow|deny/


(************************************************************************
 * Group:                 ENTRY
 *************************************************************************)

(*
View: entry
  - It might be indented with an arbitrary amount of whitespace
  - It does not have any separator between keywords and their values
  - It can only have keywords with the following values (path, allow, deny)
*)
let entry = IniFile.indented_entry entry_re sep comment


(************************************************************************
 * Group:                      RECORD
 *************************************************************************)

(* Group: Title definition *)

(*
View: title
  Uses standard INI File title
*)
let title = IniFile.indented_title IniFile.record_re

(*
View: title
  Uses standard INI File record
*)
let record = IniFile.record title entry


(************************************************************************
 * Group:                      LENS
 *************************************************************************)

(*
View: lns
  Uses standard INI File lens
*)
let lns = IniFile.lns record comment

(* Variable: filter *)
let filter = (incl "/etc/puppet/fileserver.conf"
             .incl "/usr/local/etc/puppet/fileserver.conf"
             .incl "/etc/puppetlabs/puppet/fileserver.conf")

let xfm = transform lns filter
