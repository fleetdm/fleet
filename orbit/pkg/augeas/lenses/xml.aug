(* XML lens for Augeas
   Author: Francis Giraldeau <francis.giraldeau@usherbrooke.ca>

   Reference: http://www.w3.org/TR/2006/REC-xml11-20060816/
*)

module Xml =

autoload xfm

(************************************************************************
 *                           Utilities lens
 *************************************************************************)

let dels (s:string)   = del s s
let spc               = /[ \t\r\n]+/
let osp               = /[ \t\r\n]*/
let sep_spc           = del /[ \t\r\n]+/ " "
let sep_osp           = del /[ \t\r\n]*/ ""
let sep_eq            = del /[ \t\r\n]*=[ \t\r\n]*/ "="

let nmtoken             = /[a-zA-Z:_][a-zA-Z0-9:_.-]*/
let word                = /[a-zA-Z][a-zA-Z0-9._-]*/
let char                = /.|(\r?\n)/
(* if we hide the quotes, then we can only accept single or double quotes *)
(* otherwise a put ambiguity is raised *)
let sto_dquote          = dels "\"" . store /[^"]*/ . dels "\"" (* " *)
let sto_squote          = dels "'" . store /[^']*/ . dels "'"

let comment             = [ label "#comment" .
                            dels "<!--" .
                            store /([^-]|-[^-])*/ .
                            dels "-->" ]

let pi_target           = nmtoken - /[Xx][Mm][Ll]/
let empty               = Util.empty
let del_end             = del />[\r?\n]?/ ">\n"
let del_end_simple      = dels ">"

(* This is siplified version of processing instruction
 * pi has to not start or end with a white space and the string
 * must not contain "?>". We restrict too much by not allowing any
 * "?" nor ">" in PI
 *)
let pi                  = /[^ \r\n\t]|[^ \r\n\t][^?>]*[^ \r\n\t]/

(************************************************************************
 *                            Attributes
 *************************************************************************)


let decl          = [ label "#decl" . sep_spc .
                      store /[^> \t\n\r]|[^> \t\n\r][^>\t\n\r]*[^> \t\n\r]/ ]

let decl_def (r:regexp) (b:lens) = [ dels "<" . key r .
                                     sep_spc . store nmtoken .
                                     b . sep_osp . del_end_simple ]

let elem_def      = decl_def /!ELEMENT/ decl

let enum          = "(" . osp . nmtoken . ( osp . "|" . osp . nmtoken )* . osp . ")"

let att_type      = /CDATA|ID|IDREF|IDREFS|ENTITY|ENTITIES|NMTOKEN|NMTOKENS/ |
                     enum

let id_def        = [ sep_spc . key /PUBLIC/ .
                      [ label "#literal" . sep_spc . sto_dquote ]* ] |
                    [ sep_spc . key /SYSTEM/ . sep_spc . sto_dquote ]

let notation_def  = decl_def /!NOTATION/ id_def

let att_def       = counter "att_id" .
                    [ sep_spc . seq "att_id" .
                      [ label "#name" . store word . sep_spc ] .
                      [ label "#type" . store att_type . sep_spc ] .
                      ([ key   /#REQUIRED|#IMPLIED/ ] |
                       [ label "#FIXED" . del /#FIXED[ \r\n\t]*|/ "" . sto_dquote ]) ]*

let att_list_def = decl_def /!ATTLIST/ att_def

let entity_def   =
  let literal (lbl:string) = [ sep_spc . label lbl . sto_dquote ] in
  decl_def /!ENTITY/
    ( literal "#decl"
    | [ sep_spc . key /SYSTEM/ . literal "#systemliteral" ]
    | [ sep_spc . key /PUBLIC/ . literal "#pubidliteral"
                               . literal "#systemliteral" ] )

let decl_def_item = elem_def | entity_def | att_list_def | notation_def

let decl_outer    = sep_osp . del /\[[ \n\t\r]*/ "[\n" .
                    (decl_def_item . sep_osp )* . dels "]"

(* let dtd_def       = [ sep_spc . key "SYSTEM" . sep_spc . sto_dquote ] *)

let doctype       = decl_def /!DOCTYPE/ (decl_outer|id_def)

(* General shape of an attribute
 * q   is the regexp matching the quote character for the value
 * qd  is the default quote character
 * brx is what the actual attribute value must match *)
let attval (q:regexp) (qd:string) (brx:regexp) =
  let quote = del q qd in
  let body = store brx in
  [ sep_spc . key nmtoken . sep_eq . square quote body quote ]

(* We treat attributes according to one of the following three patterns:
   attval1 : values that must be quoted with single quotes
   attval2 : values that must be quoted with double quotes
   attval3 : values that can be quoted with either *)
let attributes    =
  let attval1 = attval "'" "'" /[^']*"[^']*/ in (* " *)
  let attval2 = attval "\"" "\"" /[^"]*'[^"]*/ in
  let attval3 = attval /['"]/ "\"" /(\\\\|[^'\"])*/ in (* " *)
  [ label "#attribute" . (attval1|attval2|attval3)+ ]

let prolog        = [ label "#declaration" .
                      dels "<?xml" .
                      attributes .
                      sep_osp .
                      dels "?>" ]


(************************************************************************
 *                            Tags
 *************************************************************************)

(* we consider entities as simple text *)
let text_re   = /[^<]+/ - /([^<]*\]\]>[^<]*)/
let text      = [ label "#text" . store text_re ]
let cdata     = [ label "#CDATA" . dels "<![CDATA[" .
                  store (char* - (char* . "]]>" . char*)) . dels "]]>" ]

(* the value of nmtoken_del is always the nmtoken_key string *)
let nmtoken_key = key nmtoken
let nmtoken_del = del nmtoken "a"

let element (body:lens) =
    let h = attributes? . sep_osp . dels ">" . body* . dels "</" in
        [ dels "<" . square nmtoken_key h nmtoken_del . sep_osp . del_end ]

let empty_element = [ dels "<" . nmtoken_key . value "#empty" .
                      attributes? . sep_osp . del /\/>[\r?\n]?/ "/>\n" ]

let pi_instruction = [ dels "<?" . label "#pi" .
                       [ label "#target" . store pi_target ] .
                       [ sep_spc . label "#instruction" . store pi ]? .
                       sep_osp . del /\?>/ "?>" ]

(* Typecheck is weaker on rec lens, detected by unfolding *)
(*
let content1 = element text
let rec content2 = element (content1|text|comment)
*)

let rec content = element (text|comment|content|empty_element|pi_instruction|cdata)

(* Constraints are weaker here, but it's better than being too strict *)
let doc = (sep_osp . (prolog  | comment | doctype | pi_instruction))* .
          ((sep_osp . content) | (sep_osp . empty_element)) .
          (sep_osp . (comment | pi_instruction ))* . sep_osp

let lns = doc | Util.empty?

let filter = (incl "/etc/xml/*.xml")
    . (incl "/etc/xml/catalog")

let xfm = transform lns filter
