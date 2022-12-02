module Krb5 =

autoload xfm

let comment = Inifile.comment IniFile.comment_re "#"
let empty = Inifile.empty
let eol = Inifile.eol
let dels = Util.del_str

let indent = del /[ \t]*/ ""
let comma_or_space_sep = del /[ \t,]{1,}/ " "
let eq = del /[ \t]*=[ \t]*/ " = "
let eq_openbr = del /[ \t]*=[ \t\n]*\{[ \t]*\n/ " = {\n"
let closebr = del /[ \t]*\}/ "}"

(* These two regexps for realms and apps are not entirely true
   - strictly speaking, there's no requirement that a realm is all upper case
   and an application only uses lowercase. But it's what's used in practice.

   Without that distinction we couldn't distinguish between applications
   and realms in the [appdefaults] section.
*)

let include_re = /include(dir)?/
let realm_re = /[A-Z0-9][.a-zA-Z0-9-]*/
let realm_anycase_re = /[A-Za-z0-9][.a-zA-Z0-9-]*/
let app_re = /[a-z][a-zA-Z0-9_]*/
let name_re = /[.a-zA-Z0-9_-]+/ - include_re

let value_br = store /[^;# \t\r\n{}]+/
let value = store /[^;# \t\r\n]+/
let entry (kw:regexp) (sep:lens) (value:lens) (comment:lens)
    = [ indent . key kw . sep . value . (comment|eol) ] | comment

let subsec_entry (kw:regexp) (sep:lens) (comment:lens)
    = ( entry kw sep value_br comment ) | empty

let simple_section (n:string) (k:regexp) =
  let title = Inifile.indented_title n in
  let entry = entry k eq value comment in
    Inifile.record title entry

let record (t:string) (e:lens) =
  let title = Inifile.indented_title t in
    Inifile.record title e

let v4_name_convert (subsec:lens) = [ indent . key "v4_name_convert" .
                        eq_openbr .  subsec* . closebr . eol ]

(*
  For the enctypes this appears to be a list of the valid entries:
       c4-hmac arcfour-hmac aes128-cts rc4-hmac
       arcfour-hmac-md5 des3-cbc-sha1 des-cbc-md5 des-cbc-crc
*)
let enctype_re = /[a-zA-Z0-9-]{3,}/
let enctypes = /permitted_enctypes|default_tgs_enctypes|default_tkt_enctypes/i

(* An #eol label prevents ambiguity between "k = v1 v2" and "k = v1\n k = v2" *)
let enctype_list (nr:regexp) (ns:string) =
  indent . del nr ns . eq
    . Build.opt_list [ label ns . store enctype_re ] comma_or_space_sep
    . (comment|eol) . [ label "#eol" ]

let libdefaults =
  let option = entry (name_re - ("v4_name_convert" |enctypes)) eq value comment in
  let enctype_lists = enctype_list /permitted_enctypes/i "permitted_enctypes"
                      | enctype_list /default_tgs_enctypes/i "default_tgs_enctypes"
                      | enctype_list /default_tkt_enctypes/i "default_tkt_enctypes" in
  let subsec = [ indent . key /host|plain/ . eq_openbr .
                   (subsec_entry name_re eq comment)* . closebr . eol ] in
  record "libdefaults" (option|enctype_lists|v4_name_convert subsec)

let login =
  let keys = /krb[45]_get_tickets|krb4_convert|krb_run_aklog/
    |/aklog_path|accept_passwd/ in
    simple_section "login" keys

let appdefaults =
  let option = entry (name_re - ("realm" | "application")) eq value_br comment in
  let realm = [ indent . label "realm" . store realm_re .
                  eq_openbr . (option|empty)* . closebr . eol ] in
  let app = [ indent . label "application" . store app_re .
                eq_openbr . (realm|option|empty)* . closebr . eol] in
    record "appdefaults" (option|realm|app)

let realms =
  let simple_option = /kdc|admin_server|database_module|default_domain/
      |/v4_realm|auth_to_local(_names)?|master_kdc|kpasswd_server/
      |/admin_server|ticket_lifetime|pkinit_(anchors|identities|identity|pool)/
      |/krb524_server/ in
  let subsec_option = /v4_instance_convert/ in
  let option = subsec_entry simple_option eq comment in
  let subsec = [ indent . key subsec_option . eq_openbr .
                   (subsec_entry name_re eq comment)* . closebr . eol ] in
  let v4subsec = [ indent . key /host|plain/ . eq_openbr .
                   (subsec_entry name_re eq comment)* . closebr . eol ] in
  let realm = [ indent . label "realm" . store realm_anycase_re .
                  eq_openbr . (option|subsec|(v4_name_convert v4subsec))* .
                  closebr . eol ] in
    record "realms" (realm|comment)

let domain_realm =
  simple_section "domain_realm" name_re

let logging =
  let keys = /kdc|admin_server|default/ in
  let xchg (m:regexp) (d:string) (l:string) =
    del m d . label l in
  let xchgs (m:string) (l:string) = xchg m m l in
  let dest =
    [ xchg /FILE[=:]/ "FILE=" "file" . value ]
    |[ xchgs "STDERR" "stderr" ]
    |[ xchgs "CONSOLE" "console" ]
    |[ xchgs "DEVICE=" "device" . value ]
    |[ xchgs "SYSLOG" "syslog" .
         ([ xchgs ":" "severity" . store /[A-Za-z0-9]+/ ].
          [ xchgs ":" "facility" . store /[A-Za-z0-9]+/ ]?)? ] in
  let entry = [ indent . key keys . eq . dest . (comment|eol) ] | comment in
    record "logging" entry

let capaths =
  let realm = [ indent . key realm_re .
                  eq_openbr .
                  (entry realm_re eq value_br comment)* . closebr . eol ] in
    record "capaths" (realm|comment)

let dbdefaults =
  let keys = /database_module|ldap_kerberos_container_dn|ldap_kdc_dn/
    |/ldap_kadmind_dn|ldap_service_password_file|ldap_servers/
    |/ldap_conns_per_server/ in
    simple_section "dbdefaults" keys

let dbmodules =
  let subsec_key = /database_name|db_library|disable_last_success/
    |/disable_lockout|ldap_conns_per_server|ldap_(kdc|kadmind)_dn/
    |/ldap_(kdc|kadmind)_sasl_mech|ldap_(kdc|kadmind)_sasl_authcid/
    |/ldap_(kdc|kadmind)_sasl_authzid|ldap_(kdc|kadmind)_sasl_realm/
    |/ldap_kerberos_container_dn|ldap_servers/
    |/ldap_service_password_file|mapsize|max_readers|nosync/
    |/unlockiter/ in
  let subsec_option = subsec_entry subsec_key eq comment in
  let key = /db_module_dir/ in
  let option = entry key eq value comment in
  let realm = [ indent . label "realm" . store realm_re .
                  eq_openbr . (subsec_option)* . closebr . eol ] in
    record "dbmodules" (option|realm)

(* This section is not documented in the krb5.conf manpage,
   but the Fermi example uses it. *)
let instance_mapping =
  let value = dels "\"" . store /[^;# \t\r\n{}]*/ . dels "\"" in
  let map_node = label "mapping" . store /[a-zA-Z0-9\/*]+/ in
  let mapping = [ indent . map_node . eq .
                    [ label "value" . value ] . (comment|eol) ] in
  let instance = [ indent . key name_re .
                     eq_openbr . (mapping|comment)* . closebr . eol ] in
    record "instancemapping" instance

let kdc =
  simple_section "kdc" /profile/

let pam =
  simple_section "pam" name_re

let plugins =
  let interface_option = subsec_entry name_re eq comment in
  let interface = [ indent . key name_re .
                  eq_openbr . (interface_option)* . closebr . eol ] in
    record "plugins" (interface|comment)

let includes = Build.key_value_line include_re Sep.space (store Rx.fspath)
let include_lines = includes . (comment|empty)*

let lns = (comment|empty)* .
  (libdefaults|login|appdefaults|realms|domain_realm
  |logging|capaths|dbdefaults|dbmodules|instance_mapping|kdc|pam|include_lines
  |plugins)*

let filter = (incl "/etc/krb5.conf.d/*.conf")
           . (incl "/etc/krb5.conf")

let xfm = transform lns filter
