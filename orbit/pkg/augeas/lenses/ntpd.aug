(*
Module: Ntpd
    Parses OpenNTPD's ntpd.conf

Author: Jasper Lievisse Adriaanse <jasper@jasper.la>

About: Reference
    This lens is used to parse OpenNTPD's configuration file, ntpd.conf.
    http://openntpd.org/

About: Usage Example
    To be documented

About: License
    This file is licensed under the LGPL v2+, like the rest of Augeas.

About: Configuration files
  This lens applies to /etc/ntpd.conf.
See <filter>.
*)

module Ntpd =
autoload xfm

(************************************************************************
 * Group: Utility variables/functions
 ************************************************************************)

(* View: comment *)
let comment = Util.comment
(* View: empty *)
let empty   = Util.empty
(* View: eol *)
let eol     = Util.eol
(* View: space *)
let space   = Sep.space
(* View: word *)
let word    = Rx.word
(* View: device_re *)
let device_re = Rx.device_name | /\*/

(* View: address_re *)
let address_re = Rx.ip | /\*/ | Rx.hostname

(* View: stratum_re
   value between 1 and 15 *)
let stratum_re = /1[0-5]|[1-9]/

(* View: refid_re
   string with length < 5 *)
let refid_re = /[A-Za-z0-9_.-]{1,5}/

(* View: weight_re
   value between 1 and 10 *)
let weight_re = /10|[1-9]/

(* View: rtable_re
   0 - RT_TABLE_MAX *)
let rtable_re = Rx.byte

(* View: correction_re
   should actually only match between -127000000 and 127000000 *)
let correction_re = Rx.relinteger_noplus

(************************************************************************
 * View: key_opt_rtable_line
 *   A subnode with a keyword, an optional routing table id and an end
 *   of line.
 *
 *   Parameters:
 *     kw:regexp - the pattern to match as key
 *     sto:lens  - the storing lens
 ************************************************************************)
let key_opt_rtable_line (kw:regexp) (sto:lens) =
    let rtable = [ Util.del_str "rtable" . space . label "rtable"
                   . store rtable_re ]
      in [ key kw . space . sto . (space . rtable)? . eol ]

(************************************************************************
 * View: key_opt_weight_rtable_line
 *   A subnode with a keyword, an optional routing table id, an optional
 *   weight-value and an end of line.
 *   of line.
 *
 *   Parameters:
 *     kw:regexp - the pattern to match as key
 *     sto:lens  - the storing lens
 ************************************************************************)
let key_opt_weight_rtable_line (kw:regexp) (sto:lens) =
    let rtable = [ Util.del_str "rtable" . space . label "rtable" . store rtable_re ]
        in let weight = [ Util.del_str "weight" . space . label "weight"
                          . store weight_re ]
        in [ key kw . space . sto . (space . weight)? . (space . rtable)? . eol ]

(************************************************************************
 * View: opt_value
 *   A subnode for optional values.
 *
 *   Parameters:
 *     s:string - the option name and subtree label
 *     r:regexp  - the pattern to match as store
 ************************************************************************)
let opt_value (s:string) (r:regexp) =
  Build.key_value s space (store r)

(************************************************************************
 * Group: Keywords
 ************************************************************************)

(* View: listen
   listen on address [rtable table-id] *)
let listen =
  let addr = [ label "address" . store address_re ]
    in key_opt_rtable_line "listen on" addr

(* View: server
   server address [weight weight-value] [rtable table-id] *)
let server =
  let addr = [ label "address" . store address_re ]
    in key_opt_weight_rtable_line "server" addr

(* View: servers
   servers address [weight weight-value] [rtable table-id] *)
let servers =
  let addr = [ label "address" . store address_re ]
    in key_opt_weight_rtable_line "servers" addr

(* View: sensor
   sensor device [correction microseconds] [weight weight-value] [refid
             string] [stratum stratum-value] *)
let sensor =
  let device = [ label "device" . store device_re ]
    in let correction = opt_value "correction" correction_re
      in let weight = opt_value "weight" weight_re
        in let refid = opt_value "refid" refid_re
          in let stratum = opt_value "stratum" stratum_re
            in [ key "sensor" . space . device
	         . (space . correction)?
	         . (space . weight)?
		 . (space . refid)?
		 . (space . stratum)?
		 . eol ]

(************************************************************************
 * Group: Lens
 ************************************************************************)

(* View: keyword *)
let keyword = listen | server | servers | sensor

(* View: lns *)
let lns = ( empty | comment | keyword )*

(* View: filter *)
let filter = (incl "/etc/ntpd.conf")

let xfm = transform lns filter
