(* Spacevars module for Augeas
 Author: Free Ekanayaka <free@64studio.com>

 Reference: man interfaces
 This is a generic lens for simple key/value configuration files where
 keys and values are separated by a sequence of spaces or tabs.

*)

module Spacevars =
  autoload xfm

(************************************************************************
 *                           USEFUL PRIMITIVES
 *************************************************************************)

let comment = Util.comment
let empty   = Util.empty

(************************************************************************
 *                               ENTRIES
 *************************************************************************)


let entry = Build.key_ws_value /[A-Za-z0-9._-]+(\[[0-9]+\])?/

let flag = [ key /[A-Za-z0-9._-]+(\[[0-9]+\])?/ . Util.doseol ]

(************************************************************************
 *                                LENS
 *************************************************************************)

let lns = (comment|empty|entry|flag)*

let simple_lns = lns    (* An alias for compatibility reasons *)

(* configuration files that can be parsed without customizing the lens *)
let filter = incl "/etc/havp/havp.config"
           . incl "/etc/ldap.conf"
           . incl "/etc/ldap/ldap.conf"
           . incl "/etc/libnss-ldap.conf"
           . incl "/etc/pam_ldap.conf"

let xfm = transform lns filter
