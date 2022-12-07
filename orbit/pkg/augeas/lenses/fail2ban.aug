(* Fail2ban module for Augeas                     *)
(* Author: Nicolas Gif <ngf18490@pm.me>           *)
(* Heavily based on DPUT module by Raphael Pinson *)
(* <raphink@gmail.com>                            *)
(*                                                *)

module Fail2ban =
  autoload xfm


(************************************************************************
 * INI File settings
 *************************************************************************)
let comment  = IniFile.comment IniFile.comment_re IniFile.comment_default

let sep      = IniFile.sep IniFile.sep_re IniFile.sep_default


(************************************************************************
 * "name: value" entries, with continuations in the style of RFC 822;
 * "name=value" is also accepted
 * leading whitespace is removed from values
 *************************************************************************)
let entry = IniFile.entry IniFile.entry_re sep comment


(************************************************************************
 * sections, led by a "[section]" header
 * We can't use titles as node names here since they could contain "/"
 * We remove #comment from possible keys
 * since it is used as label for comments
 * We also remove / as first character
 * because augeas doesn't like '/' keys (although it is legal in INI Files)
 *************************************************************************)
let title   = IniFile.title IniFile.record_re
let record  = IniFile.record title entry

let lns    = IniFile.lns record comment

let filter = (incl "/etc/fail2ban/fail2ban.conf")
           . (incl "/etc/fail2ban/jail.conf")
           . (incl "/etc/fail2ban/jail.local")
           . (incl "/etc/fail2ban/fail2ban.d/*.conf")
           . (incl "/etc/fail2ban/jail.d/*.conf")

let xfm = transform lns filter

