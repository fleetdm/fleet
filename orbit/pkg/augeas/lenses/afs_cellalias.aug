(*
Module: AFS_cellalias
  Parses AFS configuration file CellAlias

Author: Pat Riehecky <riehecky@fnal.gov>

About: Reference
    This lens is targeted at the OpenAFS CellAlias file

About: Lens Usage
  Sample usage of this lens in augtool

  * Add a CellAlias for fnal.gov/files to fnal-files
  > set /files/usr/vice/etc/CellAlias/target[99] fnal.gov/files
  > set /files/usr/vice/etc/CellAlias/target[99]/linkname fnal-files

About: License
  This file is licensed under the LGPL v2+, like the rest of Augeas.
*)

module AFS_cellalias =
    autoload xfm

   (************************************************************************
    * Group:                 USEFUL PRIMITIVES
    *************************************************************************)

   (* Group: Comments and empty lines *)

   (* View: eol *)
   let eol   = Util.eol
   (* View: comment *)
   let comment = Util.comment
   (* View: empty *)
   let empty = Util.empty

   (* Group: separators *)

   (* View: space
    * Separation between key and value
    *)
   let space = Util.del_ws_spc
   let target = /[^ \t\n#]+/
   let linkname = Rx.word

   (************************************************************************
    * Group: ENTRIES
    *************************************************************************)

   (* View: entry *)
   let entry = [ label "target" . store target . space . [ label "linkname" . store linkname . eol ] ]

   (* View: lns *)
   let lns = (empty | comment | entry)*

   let xfm = transform lns (incl "/usr/vice/etc/CellAlias")

(* Local Variables: *)
(* mode: caml *)
(* End: *)
