(*
Module: Shellvars
 Generic lens for shell-script config files like the ones found
 in /etc/sysconfig

About: License
   This file is licenced under the LGPL v2+, like the rest of Augeas.

About: Lens Usage
   To be documented
*)

module Shellvars =
  autoload xfm

  (* Delete a blank line, rather than mapping it *)
  let del_empty = del (Util.empty_generic_re . "\n") "\n"

  let empty   = Util.empty
  let empty_part_re = Util.empty_generic_re . /\n+/
  let eol = del (/[ \t]+|[ \t]*[;\n]/ . empty_part_re*) "\n"
  let semicol_eol = del (/[ \t]*[;\n]/ . empty_part_re*) "\n"
  let brace_eol = del /[ \t\n]+/ "\n"

  let key_re = /[A-Za-z0-9_][-A-Za-z0-9_]*(\[[0-9A-Za-z_,]+\])?/ - ("unset" | "export")
  let matching_re = "${!" . key_re . /[\*@]\}/
  let eq = Util.del_str "="

  let eol_for_comment = del /([ \t]*\n)([ \t]*(#[ \t]*)?\n)*/ "\n"
  let comment = Util.comment_generic_seteol /[ \t]*#[ \t]*/ " # " eol_for_comment
  (* comment_eol in shell MUST begin with a space *)
  let comment_eol = Util.comment_generic_seteol /[ \t]+#[ \t]*/ " # " eol_for_comment
  let comment_or_eol = comment_eol | semicol_eol

  let xchgs   = Build.xchgs
  let semicol = del /;?/ ""

  let char  = /[^`;()'"&|\n\\# \t]#*|\\\\./
  let dquot =
       let char = /[^"\\]|\\\\./ | Rx.cl
    in "\"" . char* . "\""                    (* " Emacs, relax *)
  let squot = /'[^']*'/
  let bquot = /`[^`\n]+`/
  (* dbquot don't take spaces or semi-colons *)
  let dbquot = /``[^` \t\n;]+``/
  let dollar_assign = /\$\([^\(\)#\n]*\)/
  let dollar_arithm = /\$\(\([^\)#\n]*\)\)/

  let anyquot = (char|dquot|squot|dollar_assign|dollar_arithm)+ | bquot | dbquot
  let sto_to_semicol = store (anyquot . (Rx.cl_or_space . anyquot)*)

  (* Array values of the form '(val1 val2 val3)'. We do not handle empty *)
  (* arrays here because of typechecking headaches. Instead, they are    *)
  (* treated as a simple value                                           *)
  let array =
    let array_value = store anyquot in
    del /\([ \t]*/ "(" . counter "values" .
      [ seq "values" . array_value ] .
      [ del /[ \t\n]+/ " " . seq "values" . array_value ] *
      . del /[ \t]*\)/ ")"

  (* Treat an empty list () as a value '()'; that's not quite correct *)
  (* but fairly close.                                                *)
  let simple_value =
    let empty_array = /\([ \t]*\)/ in
      store (anyquot | empty_array)?

  let export = [ key "export" . Util.del_ws_spc ]
  let kv = Util.indent . export? . key key_re
           . eq . (simple_value | array)

  let var_action (name:string) =
    Util.indent . del name name . Util.del_ws_spc
    . label ("@" . name) . counter "var_action"
    . Build.opt_list [ seq "var_action" . store (key_re | matching_re) ] Util.del_ws_spc

  let unset = var_action "unset"
  let bare_export = var_action "export"

  let source =
    Util.indent
    . del /\.|source/ "." . label ".source"
    . Util.del_ws_spc . store /[^;=# \t\n]+/

  let shell_builtin_cmds = "ulimit" | "shift" | "exit"

  let eval =
    Util.indent . Util.del_str "eval" . Util.del_ws_spc
    . label "@eval" . store anyquot

  let alias =
    Util.indent . Util.del_str "alias" . Util.del_ws_spc
    . label "@alias" . store key_re . eq
    . [ label "value" . store anyquot ]

  let builtin =
    Util.indent . label "@builtin"
    . store shell_builtin_cmds
    . (Sep.cl_or_space
    . [ label "args" . sto_to_semicol ])?

  let keyword (kw:string) = Util.indent . Util.del_str kw
  let keyword_label (kw:string) (lbl:string) = keyword kw . label lbl

  let return =
    Util.indent . label "@return"
    . Util.del_str "return"
    . ( Util.del_ws_spc . store Rx.integer )?

  let action (operator:string) (lbl:string) (sto:lens) =
       let sp = Rx.cl_or_opt_space | /[ \t\n]+/
    in [ del (sp . operator . sp) (" " . operator . " ")
       . label ("@".lbl) . sto ]

  let action_pipe = action "|" "pipe"
  let action_and = action "&&" "and"
  let action_or = action "||" "or"

  let condition =
    let cond (start:string) (end:string) = [ label "type" . store start ]
                                         . Util.del_ws_spc . sto_to_semicol
                                         . Util.del_ws_spc . Util.del_str end
                                         . ( action_and sto_to_semicol | action_or sto_to_semicol )*
    in Util.indent . label "@condition" . (cond "[" "]" | cond "[[" "]]")

  (* Entry types *)
  let entry_eol_item (item:lens) = [ item . comment_or_eol ]
  let entry_item (item:lens) = [ item ]

  let entry_eol_nocommand =
      entry_eol_item source
        | entry_eol_item kv
        | entry_eol_item unset
        | entry_eol_item bare_export
        | entry_eol_item builtin
        | entry_eol_item return
        | entry_eol_item condition
        | entry_eol_item eval
        | entry_eol_item alias

  let entry_noeol_nocommand =
      entry_item source
        | entry_item kv
        | entry_item unset
        | entry_item bare_export
        | entry_item builtin
        | entry_item return
        | entry_item condition
        | entry_item eval
        | entry_item alias

  (* Command *)
  let rec command =
       let env = [ key key_re . eq . store anyquot . Sep.cl_or_space ]
    in let reserved_key = /exit|shift|return|ulimit|unset|export|source|\.|if|for|select|while|until|then|else|fi|done|case|eval|alias/
    in let word = /\$?[-A-Za-z0-9_.\/]+/
    in let entry_eol = entry_eol_nocommand | entry_eol_item command
    in let entry_noeol = entry_noeol_nocommand | entry_item command
    in let entry = entry_eol | entry_noeol
    in let pipe = action_pipe (entry_eol_item command | entry_item command)
    in let and = action_and entry
    in let or = action_or entry
    in Util.indent . label "@command" . env* . store (word - reserved_key)
     . [ Sep.cl_or_space . label "@arg" . sto_to_semicol]?
     . ( pipe | and | or )?

  let entry_eol = entry_eol_nocommand
                | entry_eol_item command

  let entry_noeol = entry_noeol_nocommand
                  | entry_item command

(************************************************************************
 * Group:                 CONDITIONALS AND LOOPS
 *************************************************************************)

  let generic_cond_start (start_kw:string) (lbl:string)
                         (then_kw:string) (contents:lens) =
      keyword_label start_kw lbl . Sep.space
      . sto_to_semicol
      . ( action_and sto_to_semicol | action_or sto_to_semicol )*
      . semicol_eol
      . keyword then_kw . eol
      . contents

  let generic_cond (start_kw:string) (lbl:string)
                       (then_kw:string) (contents:lens) (end_kw:string) =
      [ generic_cond_start start_kw lbl then_kw contents
        . keyword end_kw . comment_or_eol ]

  let cond_if (entry:lens) =
    let elif = [ generic_cond_start "elif" "@elif" "then" entry+ ] in
    let else = [ keyword_label "else" "@else" . eol . entry+ ] in
    generic_cond "if" "@if" "then" (entry+ . elif* . else?) "fi"

  let loop_for (entry:lens) =
    generic_cond "for" "@for" "do" entry+ "done"

  let loop_while (entry:lens) =
    generic_cond "while" "@while" "do" entry+ "done"

  let loop_until (entry:lens) =
    generic_cond "until" "@until" "do" entry+ "done"

  let loop_select (entry:lens) =
    generic_cond "select" "@select" "do" entry+ "done"

  let case (entry:lens) (entry_noeol:lens) =
       let pattern = [ label "@pattern" . sto_to_semicol . Sep.opt_space ]
    in let case_entry = [ label "@case_entry"
                       . Util.indent . pattern
                       . (Util.del_str "|" . Sep.opt_space . pattern)*
                       . Util.del_str ")" . eol
                       . entry* . entry_noeol?
                       . Util.indent . Util.del_str ";;" . eol ] in
      [ keyword_label "case" "@case" . Sep.space
        . store (char+ | ("\"" . char+ . "\""))
        . del /[ \t\n]+/ " " . Util.del_str "in" . eol
        . (empty* . comment* . case_entry)*
        . empty* . comment*
        . keyword "esac" . comment_or_eol ]

  let subshell (entry:lens) =
    [ Util.indent . label "@subshell"
    . Util.del_str "{" . brace_eol
    . entry+
    . Util.indent . Util.del_str "}" . eol ]

  let function (entry:lens) (start_kw:string) (end_kw:string) =
    [ Util.indent . label "@function"
    . del /(function[ \t]+)?/ ""
    . store Rx.word . del /[ \t]*\(\)/ "()"
    . (comment_eol|brace_eol) . Util.del_str start_kw . brace_eol
    . entry+
    . Util.indent . Util.del_str end_kw . eol ]

  let rec rec_entry =
    let entry = comment | entry_eol | rec_entry in
        cond_if entry
      | loop_for entry
      | loop_select entry
      | loop_while entry
      | loop_until entry
      | case entry entry_noeol
      | function entry "{" "}"
      | function entry "(" ")"
      | subshell entry

  let lns_norec = del_empty* . (comment | entry_eol) *

  let lns = del_empty* . (comment | entry_eol | rec_entry) *

  let sc_incl (n:string) = (incl ("/etc/sysconfig/" . n))
  let sc_excl (n:string) = (excl ("/etc/sysconfig/" . n))

  let filter_sysconfig =
      sc_incl "*" .
      sc_excl "anaconda" .
      sc_excl "bootloader" .
      sc_excl "hw-uuid" .
      sc_excl "hwconf" .
      sc_excl "ip*tables" .
      sc_excl "ip*tables.save" .
      sc_excl "kernel" .
      sc_excl "*.pub" .
      sc_excl "sysstat.ioconf" .
      sc_excl "system-config-firewall" .
      sc_excl "system-config-securitylevel" .
      sc_incl "network/config" .
      sc_incl "network/dhcp" .
      sc_incl "network/dhcp6r" .
      sc_incl "network/dhcp6s" .
      sc_incl "network/ifcfg-*" .
      sc_incl "network/if-down.d/*" .
      sc_incl "network/ifroute-*" .
      sc_incl "network/if-up.d/*" .
      sc_excl "network/if-up.d/SuSEfirewall2" .
      sc_incl "network/providers/*" .
      sc_excl "network-scripts" .
      sc_incl "network-scripts/ifcfg-*" .
      sc_excl "rhn" .
      sc_incl "rhn/allowed-actions/*" .
      sc_excl "rhn/allowed-actions/script" .
      sc_incl "rhn/allowed-actions/script/*" .
      sc_incl "rhn/rhnsd" .
      sc_excl "SuSEfirewall2.d" .
      sc_incl "SuSEfirewall2.d/cobbler" .
      sc_incl "SuSEfirewall2.d/services/*" .
      sc_excl "SuSEfirewall2.d/services/TEMPLATE" .
      sc_excl "*.systemd"

  let filter_default = incl "/etc/default/*"
                     . excl "/etc/default/grub_installdevice*"
                     . excl "/etc/default/rmt"
                     . excl "/etc/default/star"
                     . excl "/etc/default/whoopsie"
                     . incl "/etc/profile"
                     . incl "/etc/profile.d/*"
                     . excl "/etc/profile.d/*.csh"
                     . excl "/etc/profile.d/*.tcsh"
                     . excl "/etc/profile.d/csh.local"
  let filter_misc    = incl "/etc/arno-iptables-firewall/debconf.cfg"
                     . incl "/etc/conf.d/*"
                     . incl "/etc/cron-apt/config"
                     . incl "/etc/environment"
                     . incl "/etc/firewalld/firewalld.conf"
                     . incl "/etc/blkid.conf"
                     . incl "/etc/adduser.conf"
                     . incl "/etc/cowpoke.conf"
                     . incl "/etc/cvs-cron.conf"
                     . incl "/etc/cvs-pserver.conf"
                     . incl "/etc/devscripts.conf"
                     . incl "/etc/kamailio/kamctlrc"
                     . incl "/etc/lbu/lbu.conf"
                     . incl "/etc/lintianrc"
                     . incl "/etc/lsb-release"
                     . incl "/etc/os-release"
                     . incl "/etc/periodic.conf"
                     . incl "/etc/popularity-contest.conf"
                     . incl "/etc/rc.conf"
                     . incl "/etc/rc.conf.d/*"
                     . incl "/etc/rc.conf.local"
                     . incl "/etc/selinux/config"
                     . incl "/etc/ucf.conf"
                     . incl "/etc/locale.conf"
                     . incl "/etc/vconsole.conf"
                     . incl "/etc/byobu/*"

  let filter = filter_sysconfig
             . filter_default
             . filter_misc
             . Util.stdexcl

  let xfm = transform lns filter

(* Local Variables: *)
(* mode: caml       *)
(* End:             *)
