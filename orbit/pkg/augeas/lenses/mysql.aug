(* MySQL module for Augeas                    *)
(* Author: Tim Stoop <tim@kumina.nl>          *)
(* Heavily based on php.aug by Raphael Pinson *)
(* <raphink@gmail.com>                        *)
(*                                            *)

module MySQL =
  autoload xfm

(************************************************************************
 * INI File settings
 *************************************************************************)
let comment  = IniFile.comment IniFile.comment_re "#"

let sep      = IniFile.sep IniFile.sep_re IniFile.sep_default

let entry    =
     let bare = Quote.do_dquote_opt_nil (store /[^#;" \t\r\n]+([ \t]+[^#;" \t\r\n]+)*/)
  in let quoted = Quote.do_dquote (store /[^"\r\n]*[#;]+[^"\r\n]*/)
  in [ Util.indent . key IniFile.entry_re . sep . Sep.opt_space . bare . (comment|IniFile.eol) ]
   | [ Util.indent . key IniFile.entry_re . sep . Sep.opt_space . quoted . (comment|IniFile.eol) ]
   | [ Util.indent . key IniFile.entry_re . store // .  (comment|IniFile.eol) ]
   | comment

(************************************************************************
 * sections, led by a "[section]" header
 * We can't use titles as node names here since they could contain "/"
 * We remove #comment from possible keys
 * since it is used as label for comments
 * We also remove / as first character
 * because augeas doesn't like '/' keys (although it is legal in INI Files)
 *************************************************************************)
let title   = IniFile.indented_title_label "target" IniFile.record_label_re
let record  = IniFile.record title entry

let includedir = Build.key_value_line /!include(dir)?/ Sep.space (store Rx.fspath)
               . (comment|IniFile.empty)*

let lns    = (comment|IniFile.empty)* . (record|includedir)*

let filter = (incl "/etc/mysql/my.cnf")
             . (incl "/etc/mysql/conf.d/*.cnf")
             . (incl "/etc/my.cnf")
             . (incl "/etc/my.cnf.d/*.cnf")

let xfm = transform lns filter

