(* 
Module: Access
  Parses /etc/security/access.conf

Author: Lorenzo Dalrio <lorenzo.dalrio@gmail.com>

About: Reference
  Some examples of valid entries can be found in access.conf or "man access.conf"

About: License
  This file is licensed under the LGPL v2+, like the rest of Augeas.

About: Lens Usage
  Sample usage of this lens in augtool

  * Add a rule to permit login of all users from local sources (tty's, X, cron)
  > set /files/etc/security/access.conf[0] +
  > set /files/etc/security/access.conf[0]/user ALL
  > set /files/etc/security/access.conf[0]/origin LOCAL

About: Configuration files
  This lens applies to /etc/security/access.conf. See <filter>.

About: Examples
   The <Test_Access> file contains various examples and tests.
*)
module Access =
  autoload xfm

(* Group: Comments and empty lines *)
(* Variable: comment *)
let comment   = Util.comment
(* Variable: empty *)
let empty     = Util.empty

(* Group: Useful primitives *)
(* Variable: colon
 *  this is the standard field separator " : "
 *)
let colon     = del (Rx.opt_space . ":" . Rx.opt_space) " : "


(************************************************************************
 * Group:                     ENTRY LINE
  *************************************************************************)
(* View: access
 * Allow (+) or deny (-) access
 *)
let access    = label "access" . store /[+-]/

(* Variable: identifier_re
   Regex for user/group identifiers *)
let identifier_re = /[A-Za-z0-9_.\\-]+/

(* View: user_re
 * Regex for user/netgroup fields
 *)
let user_re = identifier_re - /[Ee][Xx][Cc][Ee][Pp][Tt]/

(* View: user
 * user can be a username, username@hostname or a group
 *)
let user      = [ label "user"
                . ( store user_re
                  | store Rx.word . Util.del_str "@"
                    . [ label "host" . store Rx.word ] ) ]

(* View: group
 * Format is (GROUP)
 *)
let group     = [ label "group"
                  . Util.del_str "(" . store identifier_re . Util.del_str ")" ]

(* View: netgroup
 * Format is @NETGROUP[@@NISDOMAIN]
 *)
let netgroup =
    [ label "netgroup" . Util.del_str "@" . store user_re
      . [ label "nisdomain" . Util.del_str "@@" . store Rx.word ]? ]

(* View: user_list
 * A list of users or netgroups to apply the rule to
 *)
let user_list = Build.opt_list (user|group|netgroup) Sep.space

(* View: origin_list
 * origin_list can be a single ipaddr/originname/domain/fqdn or a list of those values
 *)
let origin_list = 
   let origin_re = Rx.no_spaces - /[Ee][Xx][Cc][Ee][Pp][Tt]/
   in Build.opt_list [ label "origin" . store origin_re ] Sep.space

(* View: except
 * The except operator makes it possible to write very compact rules. 
 *)
let except (lns:lens) = [ label "except" . Sep.space
                        . del /[Ee][Xx][Cc][Ee][Pp][Tt]/ "EXCEPT"
                        . Sep.space . lns ]

(* View: entry 
 * A valid entry line
 * Definition:
 *   > entry ::= access ':' user ':' origin_list
 *)
let entry     = [ access . colon
                . user_list
                . (except user_list)?
                . colon
                . origin_list
                . (except origin_list)?
                . Util.eol ]

(************************************************************************
 * Group:                        LENS & FILTER
  *************************************************************************)
(* View: lns
    The access.conf lens, any amount of
      * <empty> lines
      * <comments>
      * <entry>
*)
let lns       = (comment|empty|entry) *

(* Variable: filter *)
let filter    = incl "/etc/security/access.conf"

(* xfm *)
let xfm       = transform lns filter
