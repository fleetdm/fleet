(* Interfaces module for Augeas
 Author: Free Ekanayaka <free@64studio.com>

 Reference: man interfaces

*)

module Interfaces =
   autoload xfm

(************************************************************************
 *                           USEFUL PRIMITIVES
 *************************************************************************)

let eol        = Util.eol

(* Define separators *)

(* a line can be extended across multiple lines by making the last  *)
(*  character a backslash *)
let sep_spc    =  del /([ \t]+|[ \t]*\\\\\n[ \t]*)/ " "

(* Define fields *)
let sto_to_eol = store /([^\\ \t\n].*[^\\ \t\n]|[^\\ \t\n])/ . eol
let sto_to_spc = store /[^\\ \t\n]+/

(* Define comments and empty lines *)

(* note that the comment definition from Util does not support *)
(* splitting lines with a backlash                             *)
let comment    = Util.comment

let empty      = Util.empty

(* Define tree stanza_ids *)
let stanza_id    (t:string) = key t . sep_spc . sto_to_spc
let stanza_param (l:string) = [ sep_spc . label l . sto_to_spc ]

(* Define reserved words and multi-value options *)
let stanza_word =
   /(source(-directory)?|iface|auto|allow-[a-z-]+|mapping|bond-slaves|bridge-ports)/

(* Define stanza option indentation *)
let stanza_indent = del /[ \t]*/ "   "

(* Define additional lines for multi-line stanzas *)
let stanza_option = [  stanza_indent
                     . key  ( /[a-z0-9_-]+/ - stanza_word )
                     . sep_spc
                     . sto_to_eol ]

(* Define space-separated array *)
let array (r:regexp) (t:string) =  del r t . label t . counter t
   . [ sep_spc . seq t . sto_to_spc ]+

(************************************************************************
 *                              AUTO
 *************************************************************************)

let auto = [ array /(allow-)?auto/ "auto" . eol ]

(************************************************************************
 *                              GENERIC ALLOW
 *************************************************************************)

let allow = [ key ( /allow-[a-z-]+/ - "allow-auto" )
             . counter "allow_seq"
             . [ sep_spc . seq "allow_seq" . sto_to_spc ]+
             . eol ]

(************************************************************************
 *                              MAPPING
 *************************************************************************)

let mapping = [ stanza_id "mapping"
               . eol
               . (stanza_option|comment|empty)+ ]

(************************************************************************
 *                              IFACE
 *************************************************************************)

let multi_option (t:string) = [ stanza_indent . array t t . eol ]

let iface   = [ Util.indent
              . stanza_id    "iface"
              . stanza_param "family"
              . stanza_param "method"
              . eol
              . ( stanza_option
                | multi_option "bond-slaves"
                | multi_option "bridge-ports"
                | comment
                | empty )* ]

(************************************************************************
 *                              SOURCE
 *************************************************************************)

let source = [ key "source" . sep_spc . sto_to_eol ]

(************************************************************************
 *                              SOURCE-DIRECTORY
 *************************************************************************)

let source_directory = [ key "source-directory" . sep_spc . sto_to_eol ]

(************************************************************************
 *                              STANZAS
 *************************************************************************)

(* The auto and hotplug stanzas always consist of one line only, while
   iface and mapping can spand along more lines. Comment nodes are
   inserted in the tree as direct children of the root node only when they
   come after an auto or hotplug stanza, otherwise they are considered part
   of an iface or mapping block *)

let stanza_single = (auto|allow|source|source_directory) . (comment|empty)*
let stanza_multi  = iface|mapping

(************************************************************************
 *                              LENS & FILTER
 *************************************************************************)

   let lns = (comment|empty)* . (stanza_multi | stanza_single)*

   let filter = (incl "/etc/network/interfaces")
                . (incl "/etc/network/interfaces.d/*")
                . Util.stdexcl

   let xfm = transform lns filter
