(*
Module: Rx
   Generic regexps to build lenses

Author: Raphael Pinson <raphink@gmail.com>

About: License
  This file is licensed under the LGPL v2+, like the rest of Augeas.
*)


module Rx =

(* Group: Spaces *)
(* Variable: space
   A mandatory space or tab *)
let space     = /[ \t]+/

(* Variable: opt_space
   An optional space or tab *)
let opt_space = /[ \t]*/

(* Variable: cl
   A continued line with a backslash *)
let cl = /[ \t]*\\\\\n[ \t]*/

(* Variable: cl_or_space
   A <cl> or a <space> *)
let cl_or_space = cl | space

(* Variable: cl_or_opt_space
   A <cl> or a <opt_space> *)
let cl_or_opt_space = cl | opt_space

(* Group: General strings *)

(* Variable: space_in
   A string which does not start or end with a space *)
let space_in  = /[^ \r\t\n].*[^ \r\t\n]|[^ \t\n\r]/

(* Variable: no_spaces
   A string with no spaces *)
let no_spaces = /[^ \t\r\n]+/

(* Variable: word
   An alphanumeric string *)
let word       = /[A-Za-z0-9_.-]+/

(* Variable: integer
   One or more digits *)
let integer    = /[0-9]+/

(* Variable: relinteger
   A relative <integer> *)
let relinteger = /[-+]?[0-9]+/

(* Variable: relinteger_noplus
   A relative <integer>, without explicit plus sign *)
let relinteger_noplus = /[-]?[0-9]+/

(* Variable: decimal
   A decimal value (using ',' or '.' as a separator) *)
let decimal    = /[0-9]+([.,][0-9]+)?/

(* Variable: reldecimal
   A relative <decimal> *)
let reldecimal    = /[+-]?[0-9]+([.,][0-9]+)?/

(* Variable: byte
  A byte (0 - 255) *)
let byte = /25[0-5]?|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9]/

(* Variable: hex
   A hex value *)
let hex = /0x[0-9a-fA-F]+/

(* Variable: octal
   An octal value *)
let octal = /0[0-7]+/

(* Variable: fspath
   A filesystem path *)
let fspath    = /[^ \t\n]+/

(* Group: All but... *)
(* Variable: neg1
   Anything but a space, a comma or a comment sign *)
let neg1      = /[^,# \n\t]+/

(*
 * Group: IPs
 * Cf. http://blog.mes-stats.fr/2008/10/09/regex-ipv4-et-ipv6/ (in fr)
 *)

(* Variable: ipv4 *)
let ipv4 =
  let dot     = "." in
    byte . dot . byte . dot . byte . dot . byte

(* Variable: ipv6 *)
let ipv6 =
  /(([0-9A-Fa-f]{1,4}:){7}[0-9A-Fa-f]{1,4})/
  | /(([0-9A-Fa-f]{1,4}:){6}:[0-9A-Fa-f]{1,4})/
  | /(([0-9A-Fa-f]{1,4}:){5}:([0-9A-Fa-f]{1,4}:)?[0-9A-Fa-f]{1,4})/
  | /(([0-9A-Fa-f]{1,4}:){4}:([0-9A-Fa-f]{1,4}:){0,2}[0-9A-Fa-f]{1,4})/
  | /(([0-9A-Fa-f]{1,4}:){3}:([0-9A-Fa-f]{1,4}:){0,3}[0-9A-Fa-f]{1,4})/
  | /(([0-9A-Fa-f]{1,4}:){2}:([0-9A-Fa-f]{1,4}:){0,4}[0-9A-Fa-f]{1,4})/
  | (    /([0-9A-Fa-f]{1,4}:){6}/
           . /((((25[0-5])|(1[0-9]{2})|(2[0-4][0-9])|([0-9]{1,2})))\.){3}/
           . /(((25[0-5])|(1[0-9]{2})|(2[0-4][0-9])|([0-9]{1,2})))/
    )
  | (    /([0-9A-Fa-f]{1,4}:){0,5}:/
           . /((((25[0-5])|(1[0-9]{2})|(2[0-4][0-9])|([0-9]{1,2})))\.){3}/
           . /(((25[0-5])|(1[0-9]{2})|(2[0-4][0-9])|([0-9]{1,2})))/
    )
  | (    /::([0-9A-Fa-f]{1,4}:){0,5}/
           . /((((25[0-5])|(1[0-9]{2})|(2[0-4][0-9])|([0-9]{1,2})))\.){3}/
           . /(((25[0-5])|(1[0-9]{2})|(2[0-4][0-9])|([0-9]{1,2})))/
    )
  | (    /[0-9A-Fa-f]{1,4}::([0-9A-Fa-f]{1,4}:){0,5}/
           . /[0-9A-Fa-f]{1,4}/
    )
  | /(::([0-9A-Fa-f]{1,4}:){0,6}[0-9A-Fa-f]{1,4})/
  | /(([0-9A-Fa-f]{1,4}:){1,7}:)/


(* Variable: ip
   An <ipv4> or <ipv6> *)
let ip        = ipv4 | ipv6


(* Variable: hostname
   A valid RFC 1123 hostname *)
let hostname = /(([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]*[a-zA-Z0-9])\.)*(
                  [A-Za-z0-9]|[A-Za-z0-9][A-Za-z0-9\-]*[A-Za-z0-9])/

(*
 * Variable: device_name
 * A Linux device name like eth0 or i2c-0. Might still be too restrictive
 *)

let device_name = /[a-zA-Z0-9_?.+:!-]+/

(*
 * Variable: email_addr
 *    To be refined
 *)
let email_addr = /[A-Za-z0-9_+.-]+@[A-Za-z0-9_.-]+/

(*
 * Variable: iso_8601
 *    ISO 8601 date time format
 *)
let year = /[0-9]{4}/
let relyear = /[-+]?/ . year
let monthday = /W?[0-9]{2}(-?[0-9]{1,2})?/
let time =
     let sep = /[T \t]/
  in let digits = /[0-9]{2}(:?[0-9]{2}(:?[0-9]{2})?)?/
  in let precis = /[.,][0-9]+/
  in let zone = "Z" | /[-+]?[0-9]{2}(:?[0-9]{2})?/
  in sep . digits . precis? . zone?
let iso_8601 = year . ("-"? . monthday . time?)?

(* Variable: url_3986
   A valid RFC 3986 url - See Appendix B *)
let url_3986 = /(([^:\/?#]+):)?(\/\/([^\/?#]*))?([^?#]*)(\?([^#]*))?(#(.*))?/
