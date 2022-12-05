(*
Module: Services
 Parses /etc/services

Author: Raphael Pinson <raphink@gmail.com>

About: Reference
 This lens tries to keep as close as possible to 'man services' where possible.

The definitions from 'man services' are put as commentaries for reference
throughout the file. More information can be found in the manual.

About: License
  This file is licensed under the LGPL v2+, like the rest of Augeas.

About: Lens Usage
  Sample usage of this lens in augtool

    * Get the name of the service running on port 22 with protocol tcp
      > match "/files/etc/services/service-name[port = '22'][protocol = 'tcp']"
    * Remove the tcp entry for "domain" service
      > rm "/files/etc/services/service-name[. = 'domain'][protocol = 'tcp']"
    * Add a tcp service named "myservice" on port 55234
      > ins service-name after /files/etc/services/service-name[last()]
      > set /files/etc/services/service-name[last()] "myservice"
      > set "/files/etc/services/service-name[. = 'myservice']/port" "55234"
      > set "/files/etc/services/service-name[. = 'myservice']/protocol" "tcp"

About: Configuration files
  This lens applies to /etc/services. See <filter>.
*)

module Services =
  autoload xfm


(************************************************************************
 * Group:                 USEFUL PRIMITIVES
 *************************************************************************)

(* Group: Generic primitives *)

(* Variable: eol *)
let eol         = del /[ \t]*(#)?[ \t]*\n/ "\n"
let indent      = Util.indent
let comment     = Util.comment
let comment_or_eol = Util.comment_or_eol
let empty       = Util.empty
let protocol_re = /[a-zA-Z]+/
let word_re     = /[a-zA-Z0-9_.+*\/:-]+/
let num_re      = /[0-9]+/

(* Group: Separators *)
let sep_spc = Util.del_ws_spc


(************************************************************************
 * Group:                 LENSES
 *************************************************************************)

(* View: port *)
let port = [ label "port" . store num_re ]

(* View: port_range *)
let port_range = [ label "start" . store num_re ]
                   . Util.del_str "-"
                   . [ label "end" . store num_re ]

(* View: protocol *)
let protocol = [ label "protocol" . store protocol_re ]

(* View: alias *)
let alias = [ label "alias" . store word_re ]

(*
 * View: record
 *   A standard /etc/services record
 *   TODO: make sure a space is added before a comment on new nodes
 *)
let record = [ label "service-name" . store word_re
                 . sep_spc . (port | port_range)
                 . del "/" "/" . protocol . ( sep_spc . alias )*
                 . comment_or_eol ]

(* View: lns
    The services lens is either <empty>, <comment> or <record> *)
let lns = ( empty | comment | record )*


(* View: filter *)
let filter = (incl "/etc/services")

let xfm = transform lns filter
