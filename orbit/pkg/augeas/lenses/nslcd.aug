(*
Module: Nslcd
  Parses /etc/nslcd.conf

Author: Jose Plana <jplana@gmail.com>

About: Reference
  This lens tries to keep as close as possible to `man 5 nslcd.conf` where
  possible.

License
   This file is licenced under the LGPL v2+, like the rest of Augeas.


About: Lens Usage

       Sample usage of this lens in augtool:

       * get uid
         > get /files/etc/nslcd.conf/threads

       * set ldap URI
         > set /files/etc/nslcd.conf/uri "ldaps://x.y.z"

       * get cache values
         > get /files/etc/nslcd.conf/cache

       * change syslog level to debug
         > set /files/etc/nslcd.conf/log "syslog debug"

       * add/change filter for the passwd map
         > set /files/etc/nslcd.conf/filter/passwd "(objectClass=posixGroup)"

       * change the default search scope
         > set /files/etc/nslcd.conf/scope[count( * )] "subtree"

       * get the default search scope
         > get /files/etc/nslcd.conf/scope[count( * )] "subtree"

       * add/set a scope search value for a specific (host) map
         > set /files/etc/nslcd.conf/scope[host]/host "subtree"

       * get all default base search
         > match /files/etc/nslcd.conf/base[count( * ) = 0]

       * get the 3rd base search default value
         > get /files/etc/nslcd.conf/base[3]

       * add a new base search default value
         > set /files/etc/nslcd.conf/base[last()+1] "dc=example,dc=com"

       * change a base search default value to a new base value
         > set /files/etc/nslcd.conf/base[self::* = "dc=example,dc=com"] "dc=test,dc=com"

       * add/change a base search for a specific map (hosts)
         > set /files/etc/nslcd.conf/base[hosts]/hosts "dc=hosts,dc=example,dc=com"

       * add a base search for a specific map (passwd)
         > set /files/etc/nslcd.conf/base[last()+1]/passwd "dc=users,dc=example,dc=com"

       * remove all base search value for a map (rpc)
         > rm /files/etc/nslcd.conf/base/rpc

       * remove a specific search base value for a map (passwd)
         > rm /files/etc/nslcd.conf/base/passwd[self::* = "dc=users,dc=example,dc=com"]

       * get an attribute mapping value for a map
         > get /files/etc/nslcd.conf/map/passwd/homeDirectory

       * get all attribute values for a map
         > match /files/etc/nslcd.conf/map/passwd/*

       * set a specific attribute for a map
         > set /files/etc/nslcd.conf/map/passwd/homeDirectory "\"${homeDirectory:-/home/$uid}\""

       * add/change a specific attribute for a map (a map that might not be defined before)
         > set /files/etc/nslcd.conf/map[shadow/userPassword]/shadow/userPassword "*"

       * remove an attribute for a specific map
         > rm /files/etc/nslcd.conf/map/shadow/userPassword

       * remove all attributes for a specific map
         > rm /files/etc/nslcd.conf/map/passwd/*

About: Configuration files
   This lens applies to /etc/nslcd.conf. See <filter>.

About: Examples
   The <Test_Nslcd> file contains various examples and tests.
*)

module Nslcd =
autoload xfm


(************************************************************************
 * Group:                 USEFUL PRIMITIVES
 *************************************************************************)


(* Group: Comments and empty lines *)

(* View: eol *)
let eol                       = Util.eol
(* View: empty *)
let empty                     = Util.empty
(* View: spc *)
let spc                       = Util.del_ws_spc
(* View: comma *)
let comma                     = Sep.comma
(* View: comment *)
let comment                   = Util.comment
(* View: do_dquote *)
let do_dquote                 = Quote.do_dquote
(* View: opt_list *)
let opt_list                  = Build.opt_list

(* Group: Ldap related values
Values that need to be parsed.
*)

(* Variable: ldap_rdn *)
let ldap_rdn                  = /[A-Za-z][A-Za-z]+=[A-Za-z0-9_.-]+/
(* Variable: ldap_dn *)
let ldap_dn                   = ldap_rdn . (/(,)?/ . ldap_rdn)*
(* Variable: ldap_filter *)
let ldap_filter               = /\(.*\)/
(* Variable: ldap_scope *)
let ldap_scope                = /sub(tree)?|one(level)?|base/
(* Variable: map_names *)
let map_names                 = /alias(es)?/
                              | /ether(s)?/
                              | /group/
                              | /host(s)?/
                              | /netgroup/
                              | /network(s)?/
                              | /passwd/
                              | /protocol(s)?/
                              | /rpc/
                              | /service(s)?/
                              | /shadow/
(* Variable: key_name *)
let key_name                  = /[^ #\n\t\/][^ #\n\t\/]+/


(************************************************************************
 * Group:                 CONFIGURATION ENTRIES
 *************************************************************************)

(* Group: Generic definitions *)

(* View: simple_entry
The simplest configuration option a key spc value. As in `gid id`
*)
let simple_entry  (kw:string) = Build.key_ws_value kw

(* View: simple_entry_quoted_value
Simple entry with quoted value
*)
let simple_entry_quoted_value (kw:string) = Build.key_value_line kw spc (do_dquote (store /.*/))

(* View simple_entry_opt_list_comma_value
Simple entry that contains a optional list separated by commas
*)
let simple_entry_opt_list_value (kw:string) (lsep:lens) = Build.key_value_line kw spc (opt_list [ seq kw . store /[^, \t\n\r]+/ ] (lsep))
(* View: key_value_line_regexp
A simple configuration option but specifying the regex for the value.
*)
let key_value_line_regexp (kw:string) (sto:regexp) = Build.key_value_line kw spc (store sto)

(* View: mapped_entry
A mapped configuration as in `filter MAP option`.
*)
let mapped_entry (kw:string) (sto:regexp)  = [ key kw . spc
                                               . Build.key_value_line map_names spc (store sto)
                                             ]
(* View: key_value_line_regexp_opt_map
A mapped configuration but the MAP value is optional as in scope [MAP] value`.
*)
let key_value_line_regexp_opt_map (kw:string) (sto:regexp) =
    ( key_value_line_regexp kw sto | mapped_entry kw sto )

(* View: map_entry
A map entry as in `map MAP ATTRIBUTE NEWATTRIBUTE`.
*)
let map_entry                 = [ key "map" . spc
                                . [ key map_names . spc
                                  . [  key key_name . spc . store Rx.no_spaces ]
                                  ] .eol
                                ]

(* Group: Option definitions *)

(* View: Base entry *)
let base_entry                = key_value_line_regexp_opt_map "base" ldap_dn

(* View: Scope entry *)
let scope_entry               = key_value_line_regexp_opt_map "scope" ldap_scope

(* View: Filter entry *)
let filter_entry              = mapped_entry "filter" ldap_filter

(* View: entries
All the combined entries.
*)
let entries                   = map_entry
                              | base_entry
                              | scope_entry
                              | filter_entry
                              | simple_entry "threads"
                              | simple_entry "uid"
                              | simple_entry "gid"
                              | simple_entry_opt_list_value "uri" spc
                              | simple_entry "ldap_version"
                              | simple_entry "binddn"
                              | simple_entry "bindpw"
                              | simple_entry "rootpwmoddn"
                              | simple_entry "rootpwmodpw"
                              | simple_entry "sasl_mech"
                              | simple_entry "sasl_realm"
                              | simple_entry "sasl_authcid"
                              | simple_entry "sasl_authzid"
                              | simple_entry "sasl_secprops"
                              | simple_entry "sasl_canonicalize"
                              | simple_entry "krb5_ccname"
                              | simple_entry "deref"
                              | simple_entry "referrals"
                              | simple_entry "bind_timelimit"
                              | simple_entry "timelimit"
                              | simple_entry "idle_timelimit"
                              | simple_entry "reconnect_sleeptime"
                              | simple_entry "reconnect_retrytime"
                              | simple_entry "ssl"
                              | simple_entry "tls_reqcert"
                              | simple_entry "tls_cacertdir"
                              | simple_entry "tls_cacertfile"
                              | simple_entry "tls_randfile"
                              | simple_entry "tls_ciphers"
                              | simple_entry "tls_cert"
                              | simple_entry "tls_key"
                              | simple_entry "pagesize"
                              | simple_entry_opt_list_value "nss_initgroups_ignoreusers" comma
                              | simple_entry "nss_min_uid"
                              | simple_entry "nss_nested_groups"
                              | simple_entry "nss_getgrent_skipmembers"
                              | simple_entry "nss_disable_enumeration"
                              | simple_entry "validnames"
                              | simple_entry "ignorecase"
                              | simple_entry "pam_authz_search"
                              | simple_entry_quoted_value "pam_password_prohibit_message"
                              | simple_entry "reconnect_invalidate"
                              | simple_entry "cache"
                              | simple_entry "log"
                              | simple_entry "pam_authc_ppolicy"

(* View: lens *)
let lns                       = (entries|empty|comment)+

(* View: filter *)
let filter                    = incl "/etc/nslcd.conf"
                              . Util.stdexcl

let xfm                       = transform lns filter
