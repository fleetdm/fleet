(*
Module: Modules_conf
  Parses /etc/modules.conf and /etc/conf.modules

  Based on the similar Modprobe lens

  Not all directives currently listed in modules.conf(5) are currently
  supported.
*)
module Modules_conf =
autoload xfm

let comment = Util.comment
let empty = Util.empty
let eol = Util.eol | Util.comment

(* Basic file structure is the same as modprobe.conf *)
let sto_to_eol = Modprobe.sto_to_eol
let sep_space = Modprobe.sep_space

let path = [ key "path" . Util.del_str "=" . sto_to_eol . eol ]
let keep = [ key "keep" . eol ]
let probeall = Build.key_value_line_comment "probeall"  sep_space
                                            sto_to_eol
                                            comment

let entry =
    Modprobe.alias
  | Modprobe.options
  | Modprobe.kv_line_command /install|pre-install|post-install/
  | Modprobe.kv_line_command /remove|pre-remove|post-remove/
  | keep
  | path
  | probeall
  

let lns = (comment|empty|entry)*

let filter = (incl "/etc/modules.conf") .
  (incl "/etc/conf.modules")

let xfm = transform lns filter
