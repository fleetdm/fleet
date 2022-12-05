(* Authinfo2 module for Augeas                    *)
(* Author: Nicolas Gif <ngf18490@pm.me>           *)
(* Heavily based on DPUT module by Raphael Pinson *)
(* <raphink@gmail.com>                            *)
(*                                                *)

module Authinfo2 =
  autoload xfm

(************************************************************************
 * INI File settings
 *************************************************************************)
let comment  = IniFile.comment IniFile.comment_re "#"

let sep      = IniFile.sep IniFile.sep_re ":"


(************************************************************************
 *                        ENTRY
 *************************************************************************)
let entry =
    IniFile.entry_generic_nocomment (key IniFile.entry_re) sep IniFile.comment_re comment


(************************************************************************
 *                         TITLE & RECORD
 *************************************************************************)
let title       = IniFile.title IniFile.record_re
let record      = IniFile.record title entry


(************************************************************************
 *                         LENS & FILTER
 *************************************************************************)
let lns    = IniFile.lns record comment

let filter = (incl (Sys.getenv("HOME") . "/.s3ql/authinfo2"))

let xfm = transform lns filter
