(*
Module: Cachefilesd
  Parses /etc/cachefilesd.conf

Author: Pat Riehecky <riehecky@fnal.gov>

About: Reference
  This lens tries to keep as close as possible to `man 5 cachefilesd.conf` where possible.

About: License
   This file is licenced under the LGPL v2+, like the rest of Augeas.

About: Lens Usage
   See <lns>.

About: Configuration files
   This lens applies to /etc/cachefilesd.conf.

About: Examples
   The <Test_Cachefilesd> file contains various examples and tests.
*)

module Cachefilesd =
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

   (* View: colon
    * Separation between selinux attributes
    *)
   let colon = Sep.colon

   (* Group: entries *)

   (* View: entry_key
    * The key for an entry in the config file
    *)
   let entry_key = Rx.word

   (* View: entry_value
    * The value for an entry may contain all sorts of things
    *)
   let entry_value = /[A-Za-z0-9_.-:%]+/

   (* View: nocull
    * The nocull key has different syntax than the rest
    *)
   let nocull = /nocull/i

   (* Group: config *)

   (* View: cacheconfig
    * This is a simple "key value" setup
    *)
   let cacheconfig = [ key (entry_key - nocull) . space
                     . store entry_value . eol ]

   (* View: nocull
    * This is a either present, and therefore active or missing and
    * not active
    *)
   let nocull_entry = [ key nocull . eol ]

  (* View: lns *)
  let lns = (empty | comment | cacheconfig | nocull_entry)*

  let xfm = transform lns (incl "/etc/cachefilesd.conf")

(* Local Variables: *)
(* mode: caml *)
(* End: *)
