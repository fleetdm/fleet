(* Rsyncd module for Augeas
   Author: Marc Fournier <marc.fournier@camptocamp.com>

  Reference: man rsyncd.conf(5)

*)

module Rsyncd =
  autoload xfm

(************************************************************************
 * INI File settings
 *************************************************************************)
let comment    = IniFile.comment IniFile.comment_re "#"
let sep        = IniFile.sep IniFile.sep_re IniFile.sep_default
let indent     = del /[ \t]*/ "  "

(* Import useful INI File primitives *)
let eol        = IniFile.eol
let empty      = IniFile.empty
let sto_to_comment
               = Util.del_opt_ws " "
               . store /[^;# \t\n][^;#\n]*[^;# \t\n]|[^;# \t\n]/


(************************************************************************
 *                        ENTRY
 * rsyncd.conf allows indented entries, but by default entries outside
 * sections are unindented
 *************************************************************************)
let entry_re   = /[A-Za-z0-9_.-][A-Za-z0-9 _.-]*[A-Za-z0-9_.-]/

let entry      = IniFile.indented_entry entry_re sep comment

(************************************************************************
 *                        RECORD & TITLE
 * We use IniFile.title_label because there can be entries
 * outside of sections whose labels would conflict with section names
 *************************************************************************)
let title   = IniFile.indented_title ( IniFile.record_re - ".anon" )
let record  = IniFile.record title entry

let record_anon = [ label ".anon" . ( entry | empty )+ ]

(************************************************************************
 *                        LENS & FILTER
 * There can be entries before any section
 * IniFile.entry includes comment management, so we just pass entry to lns
 *************************************************************************)
let lns     = record_anon? . record*

let filter = (incl "/etc/rsyncd.conf")

let xfm = transform lns filter
