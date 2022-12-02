(* Gdm module for Augeas                       *)
(* Author: Free Ekanayaka <freek@64studio.com> *)
(*                                             *)

module Gdm =
  autoload xfm

(************************************************************************
 * INI File settings
 *************************************************************************)

let comment  = IniFile.comment IniFile.comment_re IniFile.comment_default
let sep      = IniFile.sep IniFile.sep_re IniFile.sep_default
let empty    = IniFile.empty


(************************************************************************
 *                        ENTRY
 * Entry keywords can be bare digits as well (the [server] section)
 *************************************************************************)
let entry_re = ( /[A-Za-z0-9][A-Za-z0-9._-]*/ )
let entry    = IniFile.entry entry_re sep comment


(************************************************************************
 *                         TITLE
 *
 * We use IniFile.title_label because there can be entries
 * outside of sections whose labels would conflict with section names
 *************************************************************************)
let title       = IniFile.title ( IniFile.record_re - ".anon" )
let record      = IniFile.record title entry

let record_anon = [ label ".anon" . ( entry | empty )+ ]


(************************************************************************
 *                         LENS & FILTER
 * There can be entries before any section
 * IniFile.entry includes comment management, so we just pass entry to lns
 *************************************************************************)
let lns    = record_anon? . record*

let filter = (incl "/etc/gdm/gdm.conf*")
           . (incl "/etc/gdm/custom.conf")
           . Util.stdexcl

let xfm = transform lns filter
