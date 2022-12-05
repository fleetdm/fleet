(*
Module: Ldif
  Parses the LDAP Data Interchange Format (LDIF)

Author: Dominic Cleal <dcleal@redhat.com>

About: Reference
  This lens tries to keep as close as possible to RFC2849
    <http://tools.ietf.org/html/rfc2849>
  and OpenLDAP's ldif(5)

About: Licence
  This file is licensed under the LGPLv2+, like the rest of Augeas.
*)

module Ldif =
autoload xfm

(************************************************************************
 * Group:                 USEFUL PRIMITIVES
 ************************************************************************)

(* View: comment *)
let comment = Util.comment_generic /#[ \t]*/ "# "

(* View: empty
    Map empty lines, including empty comments *)
let empty   = [ del /#?[ \t]*\n/ "\n" ]

(* View: eol
    Only eol, don't include whitespace *)
let eol     = Util.del_str "\n"

(* View: sep_colon
    The separator for attributes and values *)
let sep_colon  = del /:[ \t]*/ ": "

(* View: sep_base64
    The separator for attributes and base64 encoded values *)
let sep_base64 = del /::[ \t]*/ ":: "

(* View: sep_url
    The separator for attributes and URL-sourced values *)
let sep_url    = del /:<[ \t]*/ ":< "

(* Variable: ldapoid_re
    Format of an LDAP OID from RFC 2251 *)
let ldapoid_re = /[0-9][0-9\.]*/

(* View: sep_modspec
    Separator between modify operations *)
let sep_modspec = Util.del_str "-" . eol

(************************************************************************
 * Group:                     BASIC ATTRIBUTES
 ************************************************************************)

(* Different types of values, all permitting continuation where the next line
   begins with whitespace *)
let attr_safe_string   =
     let line  = /[^ \t\n:<][^\n]*/
  in let lines = line . (/\n[ \t]+[^ \t\n][^\n]*/)*
  in sep_colon . store lines

let attr_base64_string =
     let line  = /[a-zA-Z0-9=+]+/
  in let lines = line . (/\n[ \t]+/ . line)*
  in sep_base64 . [ label "@base64" . store lines ]

let attr_url_string =
     let line  = /[^ \t\n][^\n]*/
  in let lines = line . (/\n[ \t]+/ . line)*
  in sep_url . [ label "@url" . store lines ]

let attr_intflag = sep_colon  . store /0|1/

(* View: attr_version
    version-spec = "version:" FILL version-number *)
let attr_version = Build.key_value_line "version" sep_colon (store /[0-9]+/)

(* View: attr_dn
    dn-spec = "dn:" (FILL distinguishedName /
                     ":" FILL base64-distinguishedName) *)
let attr_dn = del /dn/i "dn"
              . ( attr_safe_string | attr_base64_string )
              . eol

(* View: attr_type
    AttributeType = ldap-oid / (ALPHA *(attr-type-chars)) *)
let attr_type = ldapoid_re | /[a-zA-Z][a-zA-Z0-9-]*/
                               - /dn/i
                               - /changeType/i
                               - /include/i

(* View: attr_option
    options = option / (option ";" options) *)
let attr_option  = Util.del_str ";"
                   . [ label "@option" . store /[a-zA-Z0-9-]+/ ]

(* View: attr_description
    Attribute name, possibly with options *)
let attr_description = key attr_type . attr_option*

(* View: attr_val_spec
    Generic attribute with a value *)
let attr_val_spec = [ attr_description
                      . ( attr_safe_string
                          | attr_base64_string
                          | attr_url_string )
                      . eol ]

(* View: attr_changetype
    Parameters:
     t:regexp - value of changeType *)
let attr_changetype (t:regexp) =
  key /changeType/i . sep_colon . store t . eol

(* View: attr_modspec *)
let attr_modspec = key /add|delete|replace/ . sep_colon . store attr_type
                     . attr_option* . eol

(* View: attr_dn_value
    Parses an attribute line with a DN on the RHS
    Parameters:
     k:regexp - match attribute name as key *)
let attr_dn_value (k:regexp) =
  [ key k . ( attr_safe_string | attr_base64_string ) . eol ]

(* View: sep_line *)
let sep_line   = empty | comment

(* View: attr_include
    OpenLDAP extension, must be separated by blank lines *)
let attr_include = eol . [ key "include" . sep_colon
                     . store /[^ \t\n][^\n]*/ . eol . comment* . eol ]

(* View: sep_record *)
let sep_record = ( sep_line | attr_include )*

(************************************************************************
 * Group:                     LDIF CONTENT RECORDS
 ************************************************************************)

(* View: ldif_attrval_record
    ldif-attrval-record = dn-spec SEP 1*attrval-spec *)
let ldif_attrval_record = [ seq "record"
                            . attr_dn
                            . ( sep_line* . attr_val_spec )+ ]

(* View: ldif_content
    ldif-content = version-spec 1*(1*SEP ldif-attrval-record) *)
let ldif_content = [ label "@content"
                     . ( sep_record . attr_version )?
                     . ( sep_record . ldif_attrval_record )+
                     . sep_record ]

(************************************************************************
 * Group:                     LDIF CHANGE RECORDS
 ************************************************************************)

(* View: change_add
    change-add = "add" SEP 1*attrval-spec *)
let change_add = [ attr_changetype "add" ] . ( sep_line* . attr_val_spec )+

(* View: change_delete
    change-delete = "add" SEP 1*attrval-spec *)
let change_delete = [ attr_changetype "delete" ]

(* View: change_modspec
    change-modspec = add/delete/replace: AttributeDesc SEP *attrval-spec "-" *)
let change_modspec = attr_modspec . ( sep_line* . attr_val_spec )*

(* View: change_modify
    change-modify = "modify" SEP *mod-spec *)
let change_modify = [ attr_changetype "modify" ]
                      . ( sep_line* . [ change_modspec
                          . sep_line* . sep_modspec ] )+

(* View: change_modrdn
    ("modrdn" / "moddn") SEP newrdn/newsuperior/deleteoldrdn *)
let change_modrdn =
     let attr_deleteoldrdn = [ key "deleteoldrdn" . attr_intflag . eol ]
  in let attrs_modrdn = attr_dn_value "newrdn"
                        | attr_dn_value "newsuperior"
                        | attr_deleteoldrdn
  in [ attr_changetype /modr?dn/ ]
     . ( sep_line | attrs_modrdn )* . attrs_modrdn

(* View: change_record
    changerecord = "changetype:" FILL (changeadd/delete/modify/moddn) *)
let change_record = ( change_add | change_delete | change_modify
                      | change_modrdn)

(* View: change_control
    "control:" FILL ldap-oid 0*1(1*SPACE ("true" / "false")) 0*1(value-spec) *)
let change_control =
     let attr_criticality = [ Util.del_ws_spc . label "criticality"
                              . store /true|false/ ]
  in let attr_ctrlvalue   = [ label "value" . (attr_safe_string
                              | attr_base64_string
                              | attr_url_string ) ]
  in [ key "control" . sep_colon . store ldapoid_re
       . attr_criticality? . attr_ctrlvalue? . eol ]

(* View: ldif_change_record
    ldif-change-record = dn-spec SEP *control changerecord *)
let ldif_change_record = [ seq "record" . attr_dn
                           . ( ( sep_line | change_control )* . change_control )?
                           . sep_line* . change_record ]

(* View: ldif_changes
    ldif-changes = version-spec 1*(1*SEP ldif-change-record) *)
let ldif_changes = [ label "@changes"
                     . ( sep_record . attr_version )?
                     . ( sep_record . ldif_change_record )+
                     . sep_record ]

(************************************************************************
 * Group:                     LENS
 ************************************************************************)

(* View: lns *)
let lns = sep_record | ldif_content | ldif_changes

let filter = incl "/etc/openldap/schema/*.ldif"

let xfm = transform lns filter
