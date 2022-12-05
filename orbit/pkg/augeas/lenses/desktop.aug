(*
Module: Desktop
   Desktop module for Augeas (.desktop files)

Author: Raphael Pinson <raphink@gmail.com>

About: Lens Usage
   This lens is made to provide a lens for .desktop files for augeas

Reference: Freedesktop.org
   http://standards.freedesktop.org/desktop-entry-spec/desktop-entry-spec-latest.html

About: License
   This file is licenced under the LGPL v2+, like the rest of Augeas.
*)


module Desktop =
(* We don't load this lens by default
   Since a lot of desktop files contain unicode characters
   which we can't parse *)
(*  autoload xfm *)

(* Comments can be only of # type *)
let comment  = IniFile.comment "#" "#"


(* 	TITLE
*  These represents sections of a desktop file
*  Example : [DesktopEntry]
*)

let title = IniFile.title IniFile.record_re

let sep = IniFile.sep "=" "="

let setting = /[A-Za-z0-9_.-]+([][@A-Za-z0-9_.-]+)?/

(* Variable: sto_to_comment
Store until comment *)
let sto_to_comment = Sep.opt_space . store /[^# \t\r\n][^#\r\n]*[^# \t\r\n]|[^# \t\r\n]/

(* Entries can have comments at their end and so they are modified to represent as such *)
let entry = [ key setting . sep . sto_to_comment? . (comment|IniFile.eol) ] | comment

let record  = IniFile.record title entry

let lns    = IniFile.lns record comment

let filter = ( incl "/usr/share/applications/*.desktop"
             . incl "/usr/share/applications/screensavers/*.desktop" )

let xfm = transform lns filter
