(*
Module: Aptsources
  Parsing /etc/apt/sources.list
*)

module Aptsources =
  autoload xfm

(************************************************************************
 * Group: Utility variables/functions
 ************************************************************************)
  (* View:  sep_ws *)
  let sep_ws = Sep.space

  (* View: eol *)
  let eol = Util.del_str "\n"

  (* View: comment *)
  let comment = Util.comment
  (* View: empty *)
  let empty = Util.empty

  (* View: word *)
  let word = /[^][# \n\t]+/

  (* View: uri *)
  let uri =
       let protocol = /[a-z+]+:/
    in let path = /\/[^] \t]*/
    in let path_brack = /\[[^]]+\]\/?/
    in protocol? . path
     | protocol . path_brack

(************************************************************************
 * Group: Keywords
 ************************************************************************)
  (* View: record *)
  let record =
       let option_sep = [ label "operation" . store /[+-]/]? . Sep.equal
    in let option = Build.key_value /arch|trusted/ option_sep (store Rx.word)
    in let options = [ label "options"
                . Util.del_str "[" . Sep.opt_space
                . Build.opt_list option Sep.space
                . Sep.opt_space . Util.del_str "]"
                . sep_ws ]
    in [ Util.indent . seq "source"
       . [ label "type" . store word ] . sep_ws
       . options?
       . [ label "uri"  . store uri ] . sep_ws
       . [ label "distribution" . store word ]
       . [ label "component" . sep_ws . store word ]*
       . del /[ \t]*(#.*)?/ ""
       . eol ]

(************************************************************************
 * Group: Lens
 ************************************************************************)
  (* View: lns *)
  let lns = ( comment | empty | record ) *

  (* View: filter *)
  let filter = (incl "/etc/apt/sources.list")
      . (incl "/etc/apt/sources.list.d/*")
      . Util.stdexcl

  let xfm = transform lns filter

(* Local Variables: *)
(* mode: caml *)
(* End: *)
