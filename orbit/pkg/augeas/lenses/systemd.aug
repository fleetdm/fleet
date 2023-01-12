(*
Module: Systemd
  Parses systemd unit files.

Author: Dominic Cleal <dcleal@redhat.com>

About: Reference
  This lens tries to keep as close as possible to systemd.unit(5) and
  systemd.service(5) etc where possible.

About: License
  This file is licenced under the LGPL v2+, like the rest of Augeas.

About: Lens Usage
  To be documented

About: Configuration files
  This lens applies to /lib/systemd/system/* and /etc/systemd/system/*.
  See <filter>.
*)

module Systemd =
autoload xfm

(************************************************************************
 * Group:                 USEFUL PRIMITIVES
 *************************************************************************)

(* View: eol *)
let eol = Util.eol

(* View: eol_comment
   An <IniFile.comment> entry for standalone comment lines (; or #) *)
let comment     = IniFile.comment IniFile.comment_re "#"

(* View: eol_comment
   An <IniFile.comment> entry for end of line comments (# only) *)
let eol_comment = IniFile.comment "#" "#"

(* View: sep
   An <IniFile.sep> entry *)
let sep        = IniFile.sep "=" "="

(* Variable: entry_single_kw *)
let entry_single_kw  = "Description"

(* Variable: entry_command_kw *)
let entry_command_kw = /Exec[A-Za-z][A-Za-z0-9._-]+/

(* Variable: entry_env_kw *)
let entry_env_kw     = "Environment"

(* Variable: entry_multi_kw *)
let entry_multi_kw   =
     let forbidden = entry_single_kw | entry_command_kw | entry_env_kw
  in /[A-Za-z][A-Za-z0-9._-]+/ - forbidden

(* Variable: value_single_re *)
let value_single_re  = /[^# \t\n\\][^#\n\\]*[^# \t\n\\]|[^# \t\n\\]/

(* View: sto_value_single
   Support multiline values with a backslash *)
let sto_value_single = Util.del_opt_ws ""
                       . store (value_single_re
                                . (/\\\\\n/ . value_single_re)*)

(* View: sto_value *)
let sto_value = store /[^# \t\n]*[^# \t\n\\]/

(* Variable: value_sep
   Multi-value entries separated by whitespace or backslash and newline *)
let value_sep = del /[ \t]+|[ \t]*\\\\[ \t]*\n[ \t]*/ " "

(* Variable: value_cmd_re
   Don't parse @ and - prefix flags *)
let value_cmd_re = /[^#@ \t\n\\-][^#@ \t\n\\-][^# \t\n\\]*/

(* Variable: env_key *)
let env_key = /[A-Za-z0-9_]+(\[[0-9]+\])?/

(************************************************************************
 * Group:                 ENTRIES
 *************************************************************************)

(*
Supported entry features, selected by key names:
  * multi-value space separated attrs (the default)
  * single-value attrs (Description)
  * systemd.service: Exec* attrs with flags, command and arguments
  * systemd.service: Environment NAME=arg
*)

(* View: entry_fn
   Prototype for our various key=value lines, with optional comment *)
let entry_fn (kw:regexp) (val:lens) =
    [ key kw . sep . val . (eol_comment|eol) ]

(* View: entry_value
   Store a value that doesn't contain spaces *)
let entry_value  = [ label "value" . sto_value ]

(* View: entry_single
   Entry that takes a single value containing spaces *)
let entry_single = entry_fn entry_single_kw
                     [ label "value" . sto_value_single ]?

(* View: entry_command
   Entry that takes a space separated set of values (the default) *)
let entry_multi  = entry_fn entry_multi_kw
                     ( Util.del_opt_ws ""
                       . Build.opt_list entry_value value_sep )?

(* View: entry_command_flags
   Exec* flags "@" and "-".  Order is important, see systemd.service(8) *)
let entry_command_flags =
     let exit  = [ label "ignoreexit" . Util.del_str "-" ]
  in let arg0  = [ label "arg0" . Util.del_str "@" ]
  in exit? . arg0?

(* View: entry_command
   Entry that takes a command, arguments and the optional prefix flags *)
let entry_command =
     let cmd  = [ label "command" . store value_cmd_re ]
  in let arg  = [ seq "args" . sto_value ]
  in let args = [ counter "args" . label "arguments"
                . (value_sep . arg)+ ]
  in entry_fn entry_command_kw ( entry_command_flags . Util.del_opt_ws "" . cmd . args? )?

(* View: entry_env
   Entry that takes a space separated set of ENV=value key/value pairs *)
let entry_env =
     let envkv (env_val:lens) = key env_key . Util.del_str "=" . env_val
     (* bare has no spaces, and is optionally quoted *)
  in let bare = Quote.do_quote_opt (envkv (store /[^#'" \t\n]*[^#'" \t\n\\]/)?)
  in let bare_dqval = envkv (store /"[^#"\t\n]*"/)
  in let bare_sqval = envkv (store /'[^#'\t\n]*'/)
     (* quoted may be empty *)
  in let quoted = Quote.do_quote (envkv (store /[^#"'\n]*[ \t]+[^#"'\n]*/))
  in let envkv_quoted = [ bare ] | [ bare_dqval ] | [ bare_sqval ] | [ quoted ]
  in entry_fn entry_env_kw ( Util.del_opt_ws "" . ( Build.opt_list envkv_quoted value_sep ))


(************************************************************************
 * Group:                 LENS
 *************************************************************************)

(* View: entry
   An <IniFile.entry> *)
let entry   = entry_single | entry_multi | entry_command | entry_env | comment

(* View: include
   Includes another file at this position *)
let include = [ key ".include" . Util.del_ws_spc . sto_value
                . (eol_comment|eol) ]

(* View: title
   An <IniFile.title> *)
let title   = IniFile.title IniFile.record_re

(* View: record
   An <IniFile.record> *)
let record = IniFile.record title (entry|include)

(* View: lns
   An <IniFile.lns> *)
let lns    = IniFile.lns record (comment|include)

(* View: filter *)
let filter = incl "/lib/systemd/system/*"
           . incl "/lib/systemd/system/*/*"
           . incl "/etc/systemd/system/*"
           . incl "/etc/systemd/system/*/*"
           . incl "/etc/systemd/logind.conf"
           . incl "/etc/sysconfig/*.systemd"
           . incl "/lib/systemd/network/*"
           . incl "/usr/local/lib/systemd/network/*"
           . incl "/etc/systemd/network/*"
           . excl "/lib/systemd/system/*.d"
           . excl "/etc/systemd/system/*.d"
           . excl "/lib/systemd/system/*.wants"
           . excl "/etc/systemd/system/*.wants"
           . Util.stdexcl

let xfm = transform lns filter
