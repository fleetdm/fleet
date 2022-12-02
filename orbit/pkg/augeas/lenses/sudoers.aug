(*
Module: Sudoers
  Parses /etc/sudoers

Author: Raphael Pinson <raphink@gmail.com>

About: Reference
  This lens tries to keep as close as possible to `man sudoers` where possible.

For example, recursive definitions such as

     > Cmnd_Spec_List ::= Cmnd_Spec |
     >                    Cmnd_Spec ',' Cmnd_Spec_List

are replaced by

  >   let cmnd_spec_list = cmnd_spec . ( sep_com . cmnd_spec )*

since Augeas cannot deal with recursive definitions.
The definitions from `man sudoers` are put as commentaries for reference
throughout the file. More information can be found in the manual.

About: License
  This file is licensed under the LGPL v2+, like the rest of Augeas.


About: Lens Usage
  Sample usage of this lens in augtool

    * Set first Defaults to apply to the "LOCALNET" network alias
      > set /files/etc/sudoers/Defaults[1]/type "@LOCALNET"
    * List all user specifications applying explicitly to the "admin" Unix group
      > match /files/etc/sudoers/spec/user "%admin"
    * Remove the full 3rd user specification
      > rm /files/etc/sudoers/spec[3]

About: Configuration files
  This lens applies to /etc/sudoers. See <filter>.
*)



module Sudoers =
  autoload xfm

(************************************************************************
 * Group:                 USEFUL PRIMITIVES
 *************************************************************************)

(* Group: Generic primitives *)
(* Variable: eol *)
let eol       = Util.eol

(* Variable: indent *)
let indent    = Util.indent


(* Group: Separators *)

(* Variable: sep_spc *)
let sep_spc  = Sep.space

(* Variable: sep_cont *)
let sep_cont = Sep.cl_or_space

(* Variable: sep_cont_opt *)
let sep_cont_opt = Sep.cl_or_opt_space

(* Variable: sep_cont_opt_build *)
let sep_cont_opt_build (sep:string) =
   del (Rx.cl_or_opt_space . sep . Rx.cl_or_opt_space) (" " . sep . " ")

(* Variable: sep_com *)
let sep_com = sep_cont_opt_build ","

(* Variable: sep_eq *)
let sep_eq   = sep_cont_opt_build "="

(* Variable: sep_col *)
let sep_col  = sep_cont_opt_build ":"

(* Variable: sep_dquote *)
let sep_dquote   = Util.del_str "\""

(* Group: Negation expressions *)

(************************************************************************
 * View: del_negate
 *   Delete an even number of '!' signs
 *************************************************************************)
let del_negate = del /(!!)*/ ""

(************************************************************************
 * View: negate_node
 *   Negation of boolean values for <defaults>. Accept one optional '!'
 *   and produce a 'negate' node if there is one.
 *************************************************************************)
let negate_node = [ del "!" "!" . label "negate" ]

(************************************************************************
 * View: negate_or_value
 *   A <del_negate>, followed by either a negated key, or a key/value pair
 *************************************************************************)
let negate_or_value (key:lens) (value:lens) =
  [ del_negate . (negate_node . key | key . value) ]

(* Group: Stores *)

(* Variable: sto_to_com_cmnd
sto_to_com_cmnd does not begin or end with a space *)

let sto_to_com_cmnd = del_negate . negate_node? . (
      let alias = Rx.word - /(NO)?(PASSWD|EXEC|SETENV)/
     in let non_alias = /[\/a-z]([^,:#()\n\\]|\\\\[=:,\\])*[^,=:#() \t\n\\]|[^,=:#() \t\n\\]/
   in store (alias | non_alias))

(* Variable: sto_to_com

There could be a \ in the middle of a command *)
let sto_to_com      = store /([^,=:#() \t\n\\][^,=:#()\n]*[^,=:#() \t\n\\])|[^,=:#() \t\n\\]/

(* Variable: sto_to_com_host *)
let sto_to_com_host = store /[^,=:#() \t\n\\]+/


(* Variable: sto_to_com_user
Escaped spaces and NIS domains and allowed*)
let sto_to_com_user =
      let nis_re = /([A-Z]([-A-Z0-9]|(\\\\[ \t]))*+\\\\\\\\)/
   in let user_re = /[%+@a-z]([-A-Za-z0-9._+]|(\\\\[ \t])|\\\\\\\\[A-Za-z0-9])*/ - /@include(dir)?/
   in let alias_re = /[A-Z_]+/
   in store ((nis_re? . user_re) | alias_re)

(* Variable: to_com_chars *)
let to_com_chars        = /[^",=#() \t\n\\]+/ (* " relax emacs *)

(* Variable: to_com_dquot *)
let to_com_dquot        = /"[^",=#()\n\\]+"/ (* " relax emacs *)

(* Variable: sto_to_com_dquot *)
let sto_to_com_dquot    = store (to_com_chars|to_com_dquot)

(* Variable: sto_to_com_col *)
let sto_to_com_col      = store to_com_chars

(* Variable: sto_to_eq *)
let sto_to_eq  = store /[^,=:#() \t\n\\]+/

(* Variable: sto_to_spc *)
let sto_to_spc = store /[^", \t\n\\]+|"[^", \t\n\\]+"/

(* Variable: sto_to_spc_no_dquote *)
let sto_to_spc_no_dquote = store /[^",# \t\n\\]+/ (* " relax emacs *)

(* Variable: sto_integer *)
let sto_integer = store /-?[0-9]+/


(* Group: Comments and empty lines *)

(* View: comment
Map comments in "#comment" nodes *)
let comment =
  let sto_to_eol = store (/([^ \t\n].*[^ \t\n]|[^ \t\n])/ - /include(dir)?.*/) in
  [ label "#comment" . del /[ \t]*#[ \t]*/ "# " . sto_to_eol . eol ]

(* View: comment_eol
Requires a space before the # *)
let comment_eol = Util.comment_generic /[ \t]+#[ \t]*/ " # "

(* View: comment_or_eol
A <comment_eol> or <eol> *)
let comment_or_eol = comment_eol | (del /([ \t]+#\n|[ \t]*\n)/ "\n")

(* View: empty
Map empty lines *)
let empty   = [ del /[ \t]*#?[ \t]*\n/ "\n" ]

(* View: includedir *)
let includedir =
  [ key /(#|@)include(dir)?/ . Sep.space . store Rx.fspath . eol ]


(************************************************************************
 * Group:                                    ALIASES
 *************************************************************************)

(************************************************************************
 * View: alias_field
 *   Generic alias field to gather all Alias definitions
 *
 *   Definition:
 *     > User_Alias ::= NAME '=' User_List
 *     > Runas_Alias ::= NAME '=' Runas_List
 *     > Host_Alias ::= NAME '=' Host_List
 *     > Cmnd_Alias ::= NAME '=' Cmnd_List
 *
 *   Parameters:
 *     kw:string - the label string
 *     sto:lens  - the store lens
 *************************************************************************)
let alias_field (kw:string) (sto:lens) = [ label kw . sto ]

(* View: alias_list
     List of <alias_fields>, separated by commas *)
let alias_list  (kw:string) (sto:lens) =
  Build.opt_list (alias_field kw sto) sep_com

(************************************************************************
 * View: alias_name
 *   Name of an <alias_entry_single>
 *
 *   Definition:
 *     > NAME ::= [A-Z]([A-Z][0-9]_)*
 *************************************************************************)
let alias_name
    = [ label "name" . store /[A-Z][A-Z0-9_]*/ ]

(************************************************************************
 * View: alias_entry_single
 *   Single <alias_entry>, named using <alias_name> and listing <alias_list>
 *
 *   Definition:
 *     > Alias_Type NAME = item1, item2, ...
 *
 *   Parameters:
 *     field:string - the field name, passed to <alias_list>
 *     sto:lens     - the store lens, passed to <alias_list>
 *************************************************************************)
let alias_entry_single (field:string) (sto:lens)
    = [ label "alias" . alias_name . sep_eq . alias_list field sto ]

(************************************************************************
 * View: alias_entry
 *   Alias entry, a list of comma-separated <alias_entry_single> fields
 *
 *   Definition:
 *     > Alias_Type NAME = item1, item2, item3 : NAME = item4, item5
 *
 *   Parameters:
 *     kw:string    - the alias keyword string
 *     field:string - the field name, passed to <alias_entry_single>
 *     sto:lens     - the store lens, passed to <alias_entry_single>
 *************************************************************************)
let alias_entry (kw:string) (field:string) (sto:lens)
    = [ indent . key kw . sep_cont . alias_entry_single field sto
          . ( sep_col . alias_entry_single field sto )* . comment_or_eol ]

(* TODO: go further in user definitions *)
(* View: user_alias
     User_Alias, see <alias_field> *)
let user_alias  = alias_entry "User_Alias" "user" sto_to_com
(* View: runas_alias
     Run_Alias, see <alias_field> *)
let runas_alias = alias_entry "Runas_Alias" "runas_user" sto_to_com
(* View: host_alias
     Host_Alias, see <alias_field> *)
let host_alias  = alias_entry "Host_Alias" "host" sto_to_com
(* View: cmnd_alias
     Cmnd_Alias, see <alias_field> *)
let cmnd_alias  = alias_entry "Cmnd_Alias" "command" sto_to_com_cmnd


(************************************************************************
 * View: alias
 *   Every kind of Alias entry,
 *     see <user_alias>, <runas_alias>, <host_alias> and <cmnd_alias>
 *
 *   Definition:
 *     > Alias ::= 'User_Alias'  User_Alias (':' User_Alias)* |
 *     >           'Runas_Alias' Runas_Alias (':' Runas_Alias)* |
 *     >           'Host_Alias'  Host_Alias (':' Host_Alias)* |
 *     >           'Cmnd_Alias'  Cmnd_Alias (':' Cmnd_Alias)*
 *************************************************************************)
let alias = user_alias | runas_alias | host_alias | cmnd_alias

(************************************************************************
 * Group:                          DEFAULTS
 *************************************************************************)


(************************************************************************
 * View: default_type
 *   Type definition for <defaults>
 *
 *   Definition:
 *     > Default_Type ::= 'Defaults' |
 *     >                  'Defaults' '@' Host_List |
 *     >                  'Defaults' ':' User_List |
 *     >                  'Defaults' '!' Cmnd_List |
 *     >                  'Defaults' '>' Runas_List
 *************************************************************************)
let default_type     =
  let value = store /[@:!>][^ \t\n\\]+/ in
  [ label "type" . value ]

(************************************************************************
 * View: parameter_flag
 *   A flag parameter for <defaults>
 *
 *   Flags are implicitly boolean and can be turned off via the '!'  operator.
 *   Some integer, string and list parameters may also be used in a boolean
 *     context to disable them.
 *************************************************************************)
let parameter_flag_kw    = "always_set_home" | "authenticate" | "env_editor"
                         | "env_reset" | "fqdn" | "ignore_dot"
                         | "ignore_local_sudoers" | "insults" | "log_host"
                         | "log_year" | "long_otp_prompt" | "mail_always"
                         | "mail_badpass" | "mail_no_host" | "mail_no_perms"
                         | "mail_no_user" | "noexec" | "path_info"
                         | "passprompt_override" | "preserve_groups"
                         | "requiretty" | "root_sudo" | "rootpw" | "runaspw"
                         | "set_home" | "set_logname" | "setenv"
                         | "shell_noargs" | "stay_setuid" | "targetpw"
                         | "tty_tickets" | "visiblepw" | "closefrom_override"
                         | "closefrom_override" | "compress_io" | "fast_glob"
                         | "log_input" | "log_output" | "pwfeedback"
                         | "umask_override" | "use_pty" | "match_group_by_gid"
                         | "always_query_group_plugin"

let parameter_flag       = [ del_negate . negate_node?
                               . key parameter_flag_kw ]

(************************************************************************
 * View: parameter_integer
 *   An integer parameter for <defaults>
 *************************************************************************)
let parameter_integer_nobool_kw = "passwd_tries"

let parameter_integer_nobool    = [ key parameter_integer_nobool_kw . sep_eq
                                      . del /"?/ "" . sto_integer
                                      . del /"?/ "" ]


let parameter_integer_bool_kw   = "loglinelen" | "passwd_timeout"
                                | "timestamp_timeout" | "umask"

let parameter_integer_bool      =
  negate_or_value
    (key parameter_integer_bool_kw)
    (sep_eq . del /"?/ "" . sto_integer . del /"?/ "")

let parameter_integer           = parameter_integer_nobool
                                | parameter_integer_bool

(************************************************************************
 * View: parameter_string
 *   A string parameter for <defaults>
 *
 *   An odd number of '!' operators negate the value of the item;
 *      an even number just cancel each other out.
 *************************************************************************)
let parameter_string_nobool_kw = "badpass_message" | "editor" | "mailsub"
                               | "noexec_file" | "passprompt" | "runas_default"
                               | "syslog_badpri" | "syslog_goodpri"
                               | "timestampdir" | "timestampowner" | "secure_path"

let parameter_string_nobool    = [ key parameter_string_nobool_kw . sep_eq
                                     . sto_to_com_dquot ]

let parameter_string_bool_kw   = "exempt_group" | "lecture" | "lecture_file"
                               | "listpw" | "logfile" | "mailerflags"
                               | "mailerpath" | "mailto" | "mailfrom" 
                               | "syslog" | "verifypw"

let parameter_string_bool      =
  negate_or_value
    (key parameter_string_bool_kw)
    (sep_eq . sto_to_com_dquot)

let parameter_string           = parameter_string_nobool
                               | parameter_string_bool

(************************************************************************
 * View: parameter_lists
 *   A single list parameter for <defaults>
 *
 *   All lists can be used in a boolean context
 *   The argument may be a double-quoted, space-separated list or a single
 *      value without double-quotes.
 *   The list can be replaced, added to, deleted from, or disabled
 *      by using the =, +=, -=, and ! operators respectively.
 *   An odd number of '!' operators negate the value of the item;
 *      an even number just cancel each other out.
 *************************************************************************)
let parameter_lists_kw           = "env_check" | "env_delete" | "env_keep"
let parameter_lists_value        = [ label "var" . sto_to_spc_no_dquote ]
let parameter_lists_value_dquote = [ label "var"
                                     . del /"?/ "" . sto_to_spc_no_dquote
                                     . del /"?/ "" ]

let parameter_lists_values = parameter_lists_value_dquote
                           | ( sep_dquote . parameter_lists_value
                               . ( sep_cont . parameter_lists_value )+
                               . sep_dquote )

let parameter_lists_sep    = sep_cont_opt
                             . ( [ del "+" "+" . label "append" ]
                               | [ del "-" "-" . label "remove" ] )?
                             . del "=" "=" . sep_cont_opt

let parameter_lists        =
  negate_or_value
    (key parameter_lists_kw)
    (parameter_lists_sep . parameter_lists_values)

(************************************************************************
 * View: parameter
 *   A single parameter for <defaults>
 *
 *   Definition:
 *     > Parameter ::= Parameter '=' Value |
 *     >               Parameter '+=' Value |
 *     >               Parameter '-=' Value |
 *     >               '!'* Parameter
 *
 *     Parameters may be flags, integer values, strings, or lists.
 *
 *************************************************************************)
let parameter        = parameter_flag | parameter_integer
                     | parameter_string | parameter_lists

(************************************************************************
 * View: parameter_list
 *   A list of comma-separated <parameters> for <defaults>
 *
 *   Definition:
 *     > Parameter_List ::= Parameter |
 *     >                    Parameter ',' Parameter_List
 *************************************************************************)
let parameter_list   = parameter . ( sep_com . parameter )*

(************************************************************************
 * View: defaults
 *   A Defaults entry
 *
 *   Definition:
 *     > Default_Entry ::= Default_Type Parameter_List
 *************************************************************************)
let defaults = [ indent . key "Defaults" . default_type? . sep_cont
                   . parameter_list . comment_or_eol ]



(************************************************************************
 * Group:                     USER SPECIFICATION
 *************************************************************************)

(************************************************************************
 * View: runas_spec
 *   A runas specification for <spec>, using <alias_list> for listing
 *   users and/or groups used to run a command
 *
 *   Definition:
 *     > Runas_Spec ::= '(' Runas_List ')' |
 *     >                '(:' Runas_List ')' |
 *     >                '(' Runas_List ':' Runas_List ')'
 *************************************************************************)
let runas_spec_user       = alias_list "runas_user" sto_to_com
let runas_spec_group      = Util.del_str ":" . indent
                            . alias_list "runas_group" sto_to_com

let runas_spec_usergroup  = runas_spec_user . indent . runas_spec_group

let runas_spec = Util.del_str "("
                 . (runas_spec_user
                    | runas_spec_group
                    | runas_spec_usergroup )
                 . Util.del_str ")" . sep_cont_opt

(************************************************************************
 * View: tag_spec
 *   Tag specification for <spec>
 *
 *   Definition:
 *     > Tag_Spec ::= ('NOPASSWD:' | 'PASSWD:' | 'NOEXEC:' | 'EXEC:' |
 *     >              'SETENV:' | 'NOSETENV:')
 *************************************************************************)
let tag_spec   =
  [ label "tag" . store /(NO)?(PASSWD|EXEC|SETENV)/ . sep_col ]

(************************************************************************
 * View: cmnd_spec
 *   Command specification for <spec>,
 *     with optional <runas_spec> and any amount of <tag_specs>
 *
 *   Definition:
 *     > Cmnd_Spec ::= Runas_Spec? Tag_Spec* Cmnd
 *************************************************************************)
let cmnd_spec  =
  [ label "command" .  runas_spec? . tag_spec* . sto_to_com_cmnd ]

(************************************************************************
 * View: cmnd_spec_list
 *   A list of comma-separated <cmnd_specs>
 *
 *   Definition:
 *     > Cmnd_Spec_List ::= Cmnd_Spec |
 *     >                    Cmnd_Spec ',' Cmnd_Spec_List
 *************************************************************************)
let cmnd_spec_list = Build.opt_list cmnd_spec sep_com


(************************************************************************
 * View: spec_list
 *   Group of hosts with <cmnd_spec_list>
 *************************************************************************)
let spec_list = [ label "host_group" . alias_list "host" sto_to_com_host
                    . sep_eq . cmnd_spec_list ]

(************************************************************************
 * View: spec
 *   A user specification, listing colon-separated <spec_lists>
 *
 *   Definition:
 *     > User_Spec ::= User_List Host_List '=' Cmnd_Spec_List \
 *     >               (':' Host_List '=' Cmnd_Spec_List)*
 *************************************************************************)
let spec = [ label "spec" . indent
               . alias_list "user" sto_to_com_user . sep_cont
               . Build.opt_list spec_list sep_col
               . comment_or_eol ]


(************************************************************************
 * Group:                        LENS & FILTER
 *************************************************************************)

(* View: lns
     The sudoers lens, any amount of
       * <empty> lines
       * <comments>
       * <includedirs>
       * <aliases>
       * <defaults>
       * <specs>
*)
let lns = ( empty | comment | includedir | alias | defaults | spec  )*

(* View: filter *)
let filter = (incl "/etc/sudoers")
    . (incl "/usr/local/etc/sudoers")
    . (incl "/etc/sudoers.d/*")
    . (incl "/usr/local/etc/sudoers.d/*")
    . (incl "/opt/csw/etc/sudoers")
    . (incl "/etc/opt/csw/sudoers")
    . Util.stdexcl

let xfm = transform lns filter
