(*
Module: Trapperkeeper
  Parses Trapperkeeper configuration files

Author: Raphael Pinson <raphael.pinson@camptocamp.com>

About: License
   This file is licenced under the LGPL v2+, like the rest of Augeas.

About: Lens Usage
   To be documented

About: Configuration files
   This lens applies to Trapperkeeper webservice configuration files. See <filter>.

About: Examples
   The <Test_Trapperkeeper> file contains various examples and tests.
*)
module Trapperkeeper =

autoload xfm

(************************************************************************
 * Group:                 USEFUL PRIMITIVES
 *************************************************************************)

(* View: empty *)
let empty = Util.empty

(* View: comment *)
let comment = Util.comment

(* View: sep *)
let sep = del /[ \t]*[:=]/ ":"

(* View: sep_with_spc *)
let sep_with_spc = sep . Sep.opt_space

(************************************************************************
 * Group:       BLOCKS (FROM 1.2, FOR 0.10 COMPATIBILITY)
 *************************************************************************)

(* Variable: block_ldelim_newlines_re *)
let block_ldelim_newlines_re = /[ \t\n]+\{([ \t\n]*\n)?/

(* Variable: block_rdelim_newlines_re *)
let block_rdelim_newlines_re = /[ \t]*\}/

(* Variable: block_ldelim_newlines_default *)
let block_ldelim_newlines_default = "\n{\n"

(* Variable: block_rdelim_newlines_default *)
let block_rdelim_newlines_default = "}"

(************************************************************************
 * View: block_newline
 *   A block enclosed in brackets, with newlines forced
 *   and indentation defaulting to a tab.
 *
 *   Parameters:
 *     entry:lens - the entry to be stored inside the block.
 *                  This entry should not include <Util.empty>,
 *                  <Util.comment> or <Util.comment_noindent>,
 *                  should be indented and finish with an eol.
 ************************************************************************)
let block_newlines (entry:lens) (comment:lens) =
   del block_ldelim_newlines_re block_ldelim_newlines_default
 . ((entry | comment) . (Util.empty | entry | comment)*)?
 . del block_rdelim_newlines_re block_rdelim_newlines_default

(************************************************************************
 * Group:                 ENTRY TYPES
 *************************************************************************)

let opt_dquot (lns:lens) = del /"?/ "" . lns . del /"?/ ""

(* View: simple *)
let simple = [ Util.indent . label "@simple" . opt_dquot (store /[A-Za-z0-9_.\/-]+/) . sep_with_spc
             . [ label "@value" . opt_dquot (store /[^,"\[ \t\n]+/) ]
             . Util.eol ]

(* View: array *)
let array =
     let lbrack = Util.del_str "["
  in let rbrack = Util.del_str "]"
  in let opt_space = del /[ \t]*/ ""
  in let comma = opt_space . Util.del_str "," . opt_space
  in let elem = [ seq "elem" . opt_dquot (store /[^,"\[ \t\n]+/) ]
  in let elems = counter "elem" . Build.opt_list elem comma
  in [ Util.indent . label "@array" . store Rx.word
     . sep_with_spc . lbrack . Sep.opt_space
     . (elems . Sep.opt_space)?
     . rbrack . Util.eol ]

(* View: hash *)
let hash (lns:lens) = [ Util.indent . label "@hash" . store Rx.word . sep
               . block_newlines lns Util.comment
               . Util.eol ]


(************************************************************************
 * Group:                   ENTRY
 *************************************************************************)

(* Just for typechecking *)
let entry_no_rec = hash (simple|array)

(* View: entry *)
let rec entry = hash (entry|simple|array)

(************************************************************************
 * Group:                LENS AND FILTER
 *************************************************************************)

(* View: lns *)
let lns = (empty|comment)* . (entry . (empty|comment)*)*

(* Variable: filter *)
let filter = incl "/etc/puppetserver/conf.d/*"
           . incl "/etc/puppetlabs/puppetserver/conf.d/*"
           . Util.stdexcl

let xfm = transform lns filter
