(* Slapd module for Augeas
   Author: Free Ekanayaka <free@64studio.com>

   Reference: man slapd.conf(5), man slapd.access (5)

*)

module Slapd =
  autoload xfm

(************************************************************************
 *                           USEFUL PRIMITIVES
 *************************************************************************)

let eol         = Util.eol
let spc         = Util.del_ws_spc
let sep         = del /[ \t\n]+/ " "

let sto_to_eol  = store /([^ \t\n].*[^ \t\n]|[^ \t\n])/
let sto_to_spc  = store /[^\\# \t\n]+/

let comment     = Util.comment
let empty       = Util.empty

(************************************************************************
 *                           ACCESS TO
 *************************************************************************)

let access_re   = "access to"
let control_re  = "stop" | "continue" | "break"
let what        = [ spc . label "access"
                  . store (/[^\\# \t\n]+/ - ("by" | control_re)) ]

(* TODO: parse the control field, see man slapd.access (5) *)
let control     = [ spc . label "control" . store control_re ]
let by          = [ sep . key "by" . spc . sto_to_spc
                  . what? . control? ]

let access      = [ key access_re . spc. sto_to_spc . by+ . eol ]

(************************************************************************
 *                             GLOBAL
 *************************************************************************)

(* TODO: parse special field separately, see man slapd.conf (5) *)
let global_re   = "allow"
                | "argsfile"
                | "attributeoptions"
                | "attributetype"
                | "authz-policy"
                | "ldap"
                | "dn"
                | "concurrency"
                | "cron_max_pending"
                | "conn_max_pending_auth"
                | "defaultsearchbase"
                | "disallow"
                | "ditcontentrule"
                | "gentlehup"
                | "idletimeout"
                | "include"
                | "index_substr_if_minlen"
                | "index_substr_if_maxlen"
                | "index_substr_any_len"
                | "index_substr_any_step"
                | "localSSF"
                | "loglevel"
                | "moduleload"
                | "modulepath"
                | "objectclass"
                | "objectidentifier"
                | "password-hash"
                | "password-crypt-salt-format"
                | "pidfile"
                | "referral"
                | "replica-argsfile"
                | "replica-pidfile"
                | "replicationinterval"
                | "require"
                | "reverse-lookup"
                | "rootDSE"
                | "sasl-host"
                | "sasl-realm"
                | "sasl-secprops"
                | "schemadn"
                | "security"
                | "sizelimit"
                | "sockbuf_max_incoming "
                | "sockbuf_max_incoming_auth"
                | "threads"
                | "timelimit time"
                | "tool-threads"
                | "TLSCipherSuite"
                | "TLSCACertificateFile"
                | "TLSCACertificatePath"
                | "TLSCertificateFile"
                | "TLSCertificateKeyFile"
                | "TLSDHParamFile"
                | "TLSRandFile"
                | "TLSVerifyClient"
                | "TLSCRLCheck"
                | "backend"

let global     = Build.key_ws_value global_re

(************************************************************************
 *                             DATABASE
 *************************************************************************)

(* TODO: support all types of database backend *)
let database_hdb = "cachesize"
                | "cachefree"
                | "checkpoint"
                | "dbconfig"
                | "dbnosync"
                | "directory"
                | "dirtyread"
                | "idlcachesize"
                | "index"
                | "linearindex"
                | "lockdetect"
                | "mode"
                | "searchstack"
                | "shm_key"

let database_re = "suffix"
                | "lastmod"
                | "limits"
                | "maxderefdepth"
                | "overlay"
                | "readonly"
                | "replica uri"
                | "replogfile"
                | "restrict"
                | "rootdn"
                | "rootpw"
                | "subordinate"
                | "syncrepl rid"
                | "updatedn"
                | "updateref"
                | database_hdb

let database_entry =
     let val = Quote.double_opt
  in Build.key_value_line database_re Sep.space val

let database    = [ key "database"
                  . spc
                  . sto_to_eol
                  . eol
                  . (comment|empty|database_entry|access)* ]

(************************************************************************
 *                              LENS
 *************************************************************************)

let lns         = (comment|empty|global|access)* . (database)*

let filter      = incl "/etc/ldap/slapd.conf"
                . incl "/etc/openldap/slapd.conf"

let xfm         = transform lns filter
