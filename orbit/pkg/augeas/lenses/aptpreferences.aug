(*
Module: AptPreferences
  Apt/preferences module for Augeas

Author: Raphael Pinson <raphael.pinson@camptocamp.com>
*)

module AptPreferences =
autoload xfm

(************************************************************************
 * Group: Entries
 ************************************************************************)

(* View: colon *)
let colon        = del /:[ \t]*/ ": "

(* View: pin_gen
     A generic pin

   Parameters:
     lbl:string - the label *)
let pin_gen (lbl:string) = store lbl
                        . [ label lbl . Sep.space . store Rx.no_spaces ]

(* View: pin_keys *)
let pin_keys =
     let space_in = store /[^, \r\t\n][^,\n]*[^, \r\t\n]|[^, \t\n\r]/
  in Build.key_value /[aclnov]/ Sep.equal space_in

(* View: pin_options *)
let pin_options =
    let comma = Util.delim ","
 in store "release" . Sep.space
                    . Build.opt_list pin_keys comma

(* View: version_pin *)
let version_pin = pin_gen "version"

(* View: origin_pin *)
let origin_pin = pin_gen "origin"

(* View: pin *)
let pin =
     let pin_value = pin_options | version_pin | origin_pin
  in Build.key_value_line "Pin" colon pin_value

(* View: entries *)
let entries = Build.key_value_line ("Explanation"|"Package"|"Pin-Priority")
                                   colon (store Rx.space_in)
            | pin
            | Util.comment

(* View: record *)
let record = [ seq "record" . entries+ ]

(************************************************************************
 * Group: Lens
 ************************************************************************)

(* View: lns *)
let lns = Util.empty* . (Build.opt_list record Util.eol+ . Util.empty*)?

(* View: filter *)
let filter = incl "/etc/apt/preferences"
           . incl "/etc/apt/preferences.d/*"
           . Util.stdexcl

let xfm = transform lns filter
