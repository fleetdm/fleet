(* PHP module for Augeas                      *)
(* Author: Raphael Pinson <raphink@gmail.com> *)
(*                                            *)

module PHP =
  autoload xfm

(************************************************************************
 * INI File settings
 *************************************************************************)

let comment  = IniFile.comment IniFile.comment_re IniFile.comment_default
let sep      = IniFile.sep IniFile.sep_re IniFile.sep_default
let empty    = IniFile.empty


(************************************************************************
 *                        ENTRY
 *
 * We have to remove the keyword "section" from possible entry keywords
 * otherwise it would lead to an ambiguity with the "section" label
 * since PHP allows entries outside of sections.
 *************************************************************************)
let entry    =
     let word = IniFile.entry_re
  in let entry_re = word . ( "[" . word . "]" )? 
  in IniFile.indented_entry entry_re sep comment


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

let filter = (incl "/etc/php*/*/*.ini")
             . (incl "/etc/php/*/*/*.ini")
             . (incl "/etc/php.ini")
             . (incl "/etc/php.d/*.ini")
             (* PHPFPM Support *)
             . (incl "/etc/php*/fpm/pool.d/*.conf")
             . (incl "/etc/php/*/fpm/pool.d/*.conf")             
             (* Zend Community edition *)
             . (incl "/usr/local/zend/etc/php.ini")
             . (incl "/usr/local/zend/etc/conf.d/*.ini")
             . Util.stdexcl

let xfm = transform lns filter
