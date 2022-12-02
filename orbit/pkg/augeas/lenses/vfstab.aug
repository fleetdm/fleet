(*
Module: Vfstab
  Parses Solaris vfstab config file, based on Fstab lens

Author: Dominic Cleal <dcleal@redhat.com>

About: Reference
  See vfstab(4)

About: License
   This file is licenced under the LGPLv2+, like the rest of Augeas.

About: Lens Usage
   To be documented

About: Configuration files
   This lens applies to /etc/vfstab.

About: Examples
   The <Test_Vfstab> file contains various examples and tests.
*)

module Vfstab =
  autoload xfm

  let sep_tab = Sep.tab
  let sep_spc = Sep.space
  let comma   = Sep.comma
  let eol     = Util.eol

  let comment = Util.comment
  let empty   = Util.empty

  let file    = /[^# \t\n]+/

  let int     = Rx.integer
  let bool    = "yes" | "no"

  (* An option label can't contain comma, comment, equals, or space *)
  let optlabel = /[^,#= \n\t]+/ - "-"
  let spec    = /[^-,# \n\t][^ \n\t]*/

  let optional = Util.del_str "-"

  let comma_sep_list (l:string) =
    let value = [ label "value" . Util.del_str "=" . store Rx.neg1 ] in
      let lns = [ label l . store optlabel . value? ] in
         Build.opt_list lns comma

  let record = [ seq "mntent" .
                   [ label "spec" . store spec ] . sep_tab .
                   ( [ label "fsck" . store spec ] | optional ). sep_tab .
                   [ label "file" . store file ] . sep_tab .
                   comma_sep_list "vfstype" . sep_tab .
                   ( [ label "passno" . store int ] | optional ) . sep_spc .
                   [ label "atboot" . store bool ] . sep_tab .
                   ( comma_sep_list "opt" | optional ) .
                   eol ]

  let lns = ( empty | comment | record ) *
  let filter = incl "/etc/vfstab"

  let xfm = transform lns filter

(* Local Variables: *)
(* mode: caml *)
(* End: *)
