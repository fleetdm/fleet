(*
Module: Erlang
  Parses Erlang configuration files

Author: Raphael Pinson <raphael.pinson@camptocamp.com>

About: Reference
  This lens tries to keep as close as possible to `http://www.erlang.org/doc/man/config.html` where possible.

About: License
   This file is licenced under the LGPL v2+, like the rest of Augeas.

About: Lens Usage
   To be documented

About: Configuration files
   This lens applies to Erlang configuration files. See <filter>.

About: Examples
   The <Test_Erlang> file contains various examples and tests.
*)
module Erlang =


(* Group: Spacing Functions *)

(* View: space *)
let space = del /[ \t\n]*/ ""

(* View: lspace
     Add spaces to the left of char *)
let lspace (char:string) = del (/[ \t\n]*/ . char) char

(* View: rspace
     Add spaces to the right of char *)
let rspace (char:string) = del (char . /[ \t\n]*/ ) char

(* View: lrspace
     Add spaces to the left or right of char *)
let lrspace (char:string) = del (/[ \t\n]*/ . char . /[ \t\n]*/ ) char


(* Group: Separators *)

(* Variable: lbrace
     Left square bracket *)
let lbrace = "{"

(* Variable: rbrace
     Right square bracket *)
let rbrace = "}"

(* Variable: lbrack
     Left curly brackets *)
let lbrack = "["

(* Variable: rbrack
     Right curly brackets *)
let rbrack = "]"

(* Variable: lglob
     Left glob separator *)
let lglob = "<<\""

(* Variable: rglob
     Right glob separator *)
let rglob = "\">>"

(* Variable: comma *)
let comma = ","


(* Group: Value types *)

(* View: opt_list
     An optional list of elements, in square brackets *)
let opt_list (lns:lens) = rspace lbrack
                        . (Build.opt_list lns (lrspace comma) . space)?
                        . Util.del_str rbrack

(* View: integer
     Store a <Rx.integer> *)
let integer = store Rx.integer

(* View: decimal
     Store a decimal value *)
let decimal = store /[0-9]+(.[0-9]+)?/

(* View: quoted
     Store a quoted value *)
let quoted = Quote.do_quote (store /[^,\n}{]+/)

(* View: bare
     Store a bare <Rx.word> *)
let bare = store Rx.word

(* View: boolean
     Store a boolean value *)
let boolean = store /true|false/

(* View: path
     Store a path (<quoted>) *)
let path = quoted

(* View: glob
     Store a glob *)
let glob = Util.del_str lglob . store /[^\n"]+/ . Util.del_str rglob

(* View: make_value
     Make a "value" subnode for arrays/tuples *)
let make_value (lns:lens) = [ label "value" . lns ]


(* Group: Store types *)

(* View: value
     A single value *)
let value (kw:regexp) (sto:lens) =
  [ rspace lbrace
  . key kw
  . lrspace comma
  . sto
  . lspace rbrace ]

(* View: tuple
     A tuple of values *)
let tuple (one:lens) (two:lens) =
  [ rspace lbrace
  . label "tuple"
  . [ label "value" . one ]
  . lrspace comma
  . [ label "value" . two ]
  . lspace rbrace ]

(* View: tuple3
     A tuple of 3 values *)
let tuple3 (one:lens) (two:lens) (three:lens) =
  [ rspace lbrace
  . label "tuple"
  . [ label "value" . one ]
  . lrspace comma
  . [ label "value" . two ]
  . lrspace comma
  . [ label "value" . three ]
  . lspace rbrace ]

(* View: list
     A list of lenses *)
let list (kw:regexp) (lns:lens) =
  [ rspace lbrace
  . key kw
  . lrspace comma
  . opt_list lns
  . lspace rbrace ]

(* View: value_list
     A <list> of seq entries *)
let value_list (kw:regexp) (sto:lens) =
  list kw (make_value sto)

(* View: application *)
let application (name:regexp) (parameter:lens) =
  list name parameter

(* View: kernel_parameters
     Config parameters accepted for kernel app *)
let kernel_parameters =
    value "browser_cmd" path
  | value "dist_auto_connect" (store /never|once/)
  | value "error_logger" (store /tty|false|silent/)
  | value "net_setuptime" integer
  | value "net_ticktime" integer
  | value "shutdown_timeout" integer
  | value "sync_nodes_timeout" integer
  | value "start_dist_ac" boolean
  | value "start_boot_server" boolean
  | value "start_disk_log" boolean
  | value "start_pg2" boolean
  | value "start_timer" boolean

(* View: kernel
     Core Erlang kernel app configuration *)
let kernel = application "kernel" kernel_parameters

(* View: comment *)
let comment = Util.comment_generic /%[ \t]*/ "% "

(* View: config
     A top-level config *)
let config (app:lens) =
    (Util.empty | comment)*
  . rspace lbrack
  . Build.opt_list (kernel | app) (lrspace comma)
  . lrspace rbrack
  . Util.del_str "." . Util.eol
  . (Util.empty | comment)*
