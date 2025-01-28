(*
Module: Crypttab
  Parses /etc/crypttab from the cryptsetup package.

Author: Frédéric Lespez <frederic.lespez@free.fr>

About: Reference
  This lens tries to keep as close as possible to `man crypttab` where possible.

About: License
  This file is licensed under the LGPL v2+, like the rest of Augeas.

About: Lens Usage
  Sample usage of this lens in augtool

    * Create a new entry for an encrypted block devices
      > ins 01 after /files/etc/crypttab/*[last()]
      > set /files/etc/crypttab/01/target crypt_sda2
      > set /files/etc/crypttab/01/device /dev/sda2
      > set /files/etc/crypttab/01/password /dev/random
      > set /files/etc/crypttab/01/opt swap
    * Print the entry applying to the "/dev/sda2" device
      > print /files/etc/crypttab/01
    * Remove the entry applying to the "/dev/sda2" device
      > rm /files/etc/crypttab/*[device="/dev/sda2"]

About: Configuration files
  This lens applies to /etc/crypttab. See <filter>.
*)

module Crypttab =
  autoload xfm

  (************************************************************************
   * Group:                 USEFUL PRIMITIVES
   *************************************************************************)

  (* Group: Separators *)

  (* Variable: sep_tab *)
  let sep_tab = Sep.tab

  (* Variable: comma *)
  let comma   = Sep.comma

  (* Group: Generic primitives *)

  (* Variable: eol *)
  let eol     = Util.eol

  (* Variable: comment *)
  let comment = Util.comment

  (* Variable: empty *)
  let empty   = Util.empty

  (* Variable: word *)
  let word    = Rx.word

   (* Variable: optval *)
  let optval  = /[A-Za-z0-9\/_.:-]+/

  (* Variable: target *)
  let target  = Rx.device_name

  (* Variable: fspath *)
  let fspath  = Rx.fspath

  (* Variable: uuid *)
  let uuid = /UUID=[0-9a-f-]+/

  (************************************************************************
   * Group:                       ENTRIES
   *************************************************************************)

  (************************************************************************
   * View: comma_sep_list
   *   A comma-separated list of options (opt=value or opt)
   *************************************************************************)
  let comma_sep_list (l:string) =
    let value = [ label "value" . Util.del_str "=" . store optval ] in
      let lns = [ label l . store word . value? ] in
         Build.opt_list lns comma

  (************************************************************************
   * View: record
   *   A crypttab record
   *************************************************************************)

  let record = [ seq "entry" .
                   [ label "target" . store target ] . sep_tab .
                   [ label "device" . store (fspath|uuid) ] .
                   (sep_tab . [ label "password" . store fspath ] .
                    ( sep_tab . comma_sep_list "opt")? )?
                 . eol ]

  (*
   * View: lns
   *   The crypttab lens
   *)
  let lns = ( empty | comment | record ) *

  (* Variable: filter *)
  let filter = (incl "/etc/crypttab")

  let xfm = transform lns filter

(* coding: utf-8 *)
