(* Module Jaas *)
(* Original Author: Simon Vocella <voxsim@gmail.com> *)
(* Updated by: Steve Shipway <steve@steveshipway.org> *)
(* Changes: allow comments within Modules, allow optionless flags,  *)
(* allow options without linebreaks, allow naked true/false options *)
(* Trailing ';' terminator should not be included in option value   *)
(* Note: requires latest Util.aug for multiline comments to work    *)

module Jaas =

autoload xfm

let space_equal = del (/[ \t]*/ . "=" . /[ \t]*/) (" = ")
let lbrace = del (/[ \t\n]*\{[ \t]*\n/) " {\n"
let rbrace = del (/[ \t]*}[ \t]*;/) " };"
let word = /[A-Za-z0-9_.-]+/
let wsnl = del (/[ \t\n]+/) ("\n")
let endflag = del ( /[ \t]*;/ ) ( ";" )

let value_re =
        let value_squote = /'[^\n']*'/
        in let value_dquote = /"[^\n"]*"/
        in let value_tf = /(true|false)/
        in value_squote | value_dquote | value_tf

let moduleOption = [  wsnl . key word . space_equal . (store value_re) ]
let moduleSuffix = ( moduleOption  | Util.eol . Util.comment_c_style | Util.comment_multiline  )
let flag = [ Util.del_ws_spc . label "flag" . (store word) . moduleSuffix* . endflag ]
let loginModuleClass = [( Util.del_opt_ws "" . label "loginModuleClass" . (store word) . flag ) ]

let content = (Util.empty | Util.comment_c_style | Util.comment_multiline | loginModuleClass)*
let loginModule = [Util.del_opt_ws "" . label "login" . (store word . lbrace) . (content . rbrace)]

let lns = (Util.empty | Util.comment_c_style | Util.comment_multiline | loginModule)*
let filter = incl "/opt/shibboleth-idp/conf/login.config"
let xfm = transform lns filter
