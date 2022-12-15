(*
Module: Channels
  Parses channels.conf files

Author: Raphael Pinson <raphink@gmail.com>

About: Reference
  See http://linuxtv.org/vdrwiki/index.php/Syntax_of_channels.conf

About: License
   This file is licenced under the LGPL v2+, like the rest of Augeas.

About: Lens Usage
   To be documented

About: Configuration files
   This lens applies to channels.conf files.

About: Examples
   The <Test_Channels> file contains various examples and tests.
*)

module Channels =

(************************************************************************
 * Group:                 USEFUL PRIMITIVES
 *************************************************************************)

(* View: eol *)
let eol = Util.eol

(* View: comment *)
let comment = Util.comment_generic /;[ \t]*/ "; "


(* View: equal *)
let equal = Sep.equal

(* View: colon *)
let colon = Sep.colon

(* View: comma *)
let comma = Sep.comma

(* View: semicol *)
let semicol = Util.del_str ";"

(* View: plus *)
let plus = Util.del_str "+"

(* View: arroba *)
let arroba = Util.del_str "@"

(* View: no_colon *)
let no_colon = /[^: \t\n][^:\n]*[^: \t\n]|[^:\n]/

(* View: no_semicolon *)
let no_semicolon = /[^;\n]+/


(************************************************************************
 * Group:                 FUNCTIONS
 *************************************************************************)

(* View: field
   A generic field *)
let field (name:string) (sto:regexp) = [ label name . store sto ]

(* View: field_no_colon
   A <field> storing <no_colon> *)
let field_no_colon (name:string) = field name no_colon

(* View: field_int
   A <field> storing <Rx.integer> *)
let field_int (name:string) = field name Rx.integer

(* View: field_word
   A <field> storing <Rx.word> *)
let field_word (name:string) = field name Rx.word


(************************************************************************
 * Group:                 ENTRIES
 *************************************************************************)

(* View: vpid *)
let vpid =
   let codec = 
           [ equal . label "codec" . store Rx.integer ]
   in let vpid_entry (lbl:string) =
           [ label lbl . store Rx.integer . codec? ]
   in vpid_entry "vpid"
    . ( plus . vpid_entry "vpid_pcr" )?


(* View: langs *)
let langs =
   let lang =
           [ label "lang" . store Rx.word ]
   in Build.opt_list lang plus


(* View: apid *)
let apid =
   let codec =
           [ arroba . label "codec" . store Rx.integer ]
   in let options =
           equal . ( (langs . codec?) | codec )
   in let apid_entry (lbl:string) =
           [ label lbl . store Rx.integer . options? ]
   in Build.opt_list (apid_entry "apid") comma
    . ( semicol
      . Build.opt_list (apid_entry "apid_dolby") comma )?
  
(* View: tpid *)
let tpid =
   let tpid_bylang =
           [ label "tpid_bylang" . store Rx.integer
           . (equal . langs)? ]
   in field_int "tpid"
      . ( semicol . Build.opt_list tpid_bylang comma )?

(* View: caid *)
let caid =
   let caid_entry =
           [ label "caid" . store Rx.word ]
   in Build.opt_list caid_entry comma

(* View: entry *)
let entry = [ label "entry" . store no_semicolon
             . (semicol . field_no_colon "provider")? . colon
             . field_int "frequency" . colon
             . field_word "parameter" . colon
             . field_word "signal_source" . colon
             . field_int "symbol_rate" . colon
             . vpid . colon
             . apid . colon
             . tpid . colon
             . caid . colon
             . field_int "sid" . colon
             . field_int "nid" . colon
             . field_int "tid" . colon
             . field_int "rid" . eol ]

(* View: entry_or_comment *)
let entry_or_comment = entry | comment

(* View: group *)
let group =
      [ Util.del_str ":" . label "group"
      . store no_colon . eol
      . entry_or_comment* ]

(* View: lns *)
let lns = entry_or_comment* . group*
