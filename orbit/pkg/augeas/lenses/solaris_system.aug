(*
Module: Solaris_System
  Parses /etc/system on Solaris

Author: Dominic Cleal <dcleal@redhat.com>

About: Reference
  This lens tries to keep as close as possible to `man 4 system` where possible.

About: Licence
  This file is licensed under the LGPL v2+, like the rest of Augeas.

About: Lens Usage

About: Configuration files
  This lens applies to /etc/system on Solaris. See <filter>.
*)

module Solaris_System =
autoload xfm

(************************************************************************
 * Group:                 USEFUL PRIMITIVES
 ************************************************************************)

(* View: comment *)
let comment = Util.comment_generic /[ \t]*\*[ \t]*/ "* "

(* View: empty
    Map empty lines, including empty asterisk comments *)
let empty   = [ del /[ \t]*\*?[ \t]*\n/ "\n" ]

(* View: sep_colon
    The separator for key/value entries *)
let sep_colon = del /:[ \t]*/ ": "

(* View: sep_moddir
    The separator of directories in a moddir search path *)
let sep_moddir = del /[: ]+/ " "

(* View: modpath
    Individual moddir search path entry *)
let modpath = [ seq "modpath" . store /[^ :\t\n]+/ ]

(* Variable: set_operator
    Valid set operators: equals, bitwise AND and OR *)
let set_operators = /[=&|]/

(* View: set_value
    Sets an integer value or char pointer *)
let set_value = [ label "value" . store Rx.no_spaces ]

(************************************************************************
 * Group:                     COMMANDS
 ************************************************************************)

(* View: cmd_kv
    Function for simple key/value setting commands such as rootfs *)
let cmd_kv (cmd:string) (value:regexp) =
    Build.key_value_line cmd sep_colon (store value)

(* View: cmd_moddir
    The moddir command for specifying module search paths *)
let cmd_moddir =
    Build.key_value_line "moddir" sep_colon
        (Build.opt_list modpath sep_moddir)

(* View: set_var
    Loads the variable name from a set command, no module *)
let set_var = [ label "variable" . store Rx.word ]

(* View: set_var
    Loads the module and variable names from a set command *)
let set_varmod = [ label "module" . store Rx.word ]
                 . Util.del_str ":" . set_var

(* View: set_sep_spc *)
let set_sep_spc = Util.del_opt_ws " "

(* View: cmd_set
    The set command for individual kernel/module parameters *)
let cmd_set = [ key "set"
              . Util.del_ws_spc
              . ( set_var | set_varmod )
              . set_sep_spc
              . [ label "operator" . store set_operators ]
              . set_sep_spc
              . set_value
              . Util.eol ]

(************************************************************************
 * Group:                     LENS
 ************************************************************************)

(* View: lns *)
let lns = ( empty
          | comment
          | cmd_moddir
          | cmd_kv "rootdev" Rx.fspath
          | cmd_kv "rootfs" Rx.word
          | cmd_kv "exclude" Rx.fspath
          | cmd_kv "include" Rx.fspath
          | cmd_kv "forceload" Rx.fspath
          | cmd_set )*

(* Variable: filter *)
let filter = (incl "/etc/system")

let xfm = transform lns filter
