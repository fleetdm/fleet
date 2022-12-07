(*
Module: Rsyslog
  Parses /etc/rsyslog.conf

Author: Raphael Pinson <raphael.pinsons@camptocamp.com>

About: Reference
  This lens tries to keep as close as possible to `man 5 rsyslog.conf` where possible.

About: License
   This file is licenced under the LGPL v2+, like the rest of Augeas.

About: Lens Usage
   To be documented

About: Configuration files
   This lens applies to /etc/rsyslog.conf. See <filter>.

About: Examples
   The <Test_Rsyslog> file contains various examples and tests.
*)
module Rsyslog =

autoload xfm

let macro_rx = /[^,# \n\t][^#\n]*[^,# \n\t]|[^,# \n\t]/
let macro = [ key /$[A-Za-z0-9]+/ . Sep.space . store macro_rx . Util.comment_or_eol ]

let config_object_param = [ key /[A-Za-z.]+/ . Sep.equal . Quote.dquote
                            . store /[^"]+/ . Quote.dquote ]
(* Inside config objects, we allow embedded comments; we don't surface them
 * in the tree though *)
let config_sep = del /[ \t]+|[ \t]*#.*\n[ \t]*/ " "

let config_object =
  [ key /action|global|input|module|parser|timezone|include/ .
    Sep.lbracket .
    config_object_param . ( config_sep . config_object_param )* .
    Sep.rbracket . Util.comment_or_eol ]

(* View: users
   Map :omusrmsg: and a list of users, or a single *
*)
let omusrmsg = Util.del_str ":omusrmsg:" .
                 Syslog.label_opt_list_or "omusrmsg" (store Syslog.word)
                                          Syslog.comma "*"

(* View: file_tmpl
   File action with a specified template *)
let file_tmpl = Syslog.file . [ label "template" . Util.del_str ";" . store Rx.word ]

let dynamic = [ Util.del_str "?" . label "dynamic" . store Rx.word ]

let namedpipe = Syslog.pipe . Sep.space . [ label "pipe" . store Syslog.file_r ]

let action = Syslog.action | omusrmsg | file_tmpl | dynamic | namedpipe

(* Cannot use syslog program because rsyslog does not suppport #! *)
let program = [ label "program" . Syslog.bang .
    ( Syslog.opt_plus | [ Build.xchgs "-" "reverse" ] ) .
    Syslog.programs . Util.eol .  Syslog.entries ]

(* Cannot use syslog hostname because rsyslog does not suppport #+/- *)
let hostname = [ label "hostname" .
      ( Syslog.plus | [ Build.xchgs "-" "reverse" ] ) .
      Syslog.hostnames . Util.eol .  Syslog.entries ]

(* View: actions *)
let actions =
     let prop_act  = [ label "action" . action ]
  in let act_sep = del /[ \t]*\n&[ \t]*/ "\n& "
  in Build.opt_list prop_act act_sep

(* View: entry
   An entry contains selectors and an action
*)
let entry = [ label "entry" . Syslog.selectors . Syslog.sep_tab .
              actions . Util.eol ]

(* View: prop_filter
   Parses property-based filters, which start with ":" and the property name *)
let prop_filter =
     let sep = Sep.comma . Util.del_opt_ws " "
  in let prop_name = [ Util.del_str ":" . label "property" . store Rx.word ]
  in let prop_oper = [ label "operation" . store /[A-Za-z!-]+/ ]
  in let prop_val  = [ label "value" . Quote.do_dquote (store /[^\n"]*/) ]
  in [ label "filter" . prop_name . sep . prop_oper . sep . prop_val .
       Sep.space . actions . Util.eol ]

let entries = ( Syslog.empty | Util.comment | entry | macro | config_object | prop_filter )*

let lns = entries . ( program | hostname )*

let filter = incl "/etc/rsyslog.conf"
           . incl "/etc/rsyslog.d/*"
           . Util.stdexcl

let xfm = transform lns filter
