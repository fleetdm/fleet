(*
Module: Up2date
  Parses /etc/sysconfig/rhn/up2date

Author: Raphael Pinson <raphink@gmail.com>

About: Reference
  This lens tries to keep as close as possible to `man 5 up2date` where possible.

About: License
   This file is licenced under the LGPLv2+, like the rest of Augeas.

About: Lens Usage
   To be documented

About: Configuration files
   This lens applies to /etc/sysconfig/rhn/up2date. See <filter>.

About: Examples
   The <Test_Up2date> file contains various examples and tests.
*)

module Up2date =

autoload xfm

(************************************************************************
 * Group:                 USEFUL PRIMITIVES
 *************************************************************************)

(* Variable: key_re *)
let key_re   = /[^=# \t\n]+/

(* Variable: value_re *)
let value_re = /[^ \t\n;][^\n;]*[^ \t\n;]|[^ \t\n;]/

(* View: sep_semi *)
let sep_semi = Sep.semicolon

(************************************************************************
 * Group:                 ENTRIES
 *************************************************************************)

(* View: single_entry
   key=foo *)
let single_entry = [ label "value" . store value_re ]

(* View: multi_empty
   key=; *)
let multi_empty  = sep_semi

(* View: multi_value
   One value in a list setting *)
let multi_value  = [ seq "multi" . store value_re ]

(* View: multi_single
   key=foo;  (parsed as a list) *)
let multi_single = multi_value . sep_semi

(* View: multi_values
   key=foo;bar
   key=foo;bar; *)
let multi_values = multi_value . ( sep_semi . multi_value )+ . del /;?/ ";"

(* View: multi_entry
   List settings go under a 'values' node *)
let multi_entry  = [ label "values" . counter "multi"
                     . ( multi_single | multi_values | multi_empty ) ]

(* View: entry *)
let entry = [ seq "entry" . store key_re . Sep.equal
              . ( multi_entry | single_entry )? . Util.eol ]

(************************************************************************
 * Group:                 LENS
 *************************************************************************)

(* View: lns *)
let lns = (Util.empty | Util.comment | entry)*

(* Variable: filter *)
let filter = incl "/etc/sysconfig/rhn/up2date"

let xfm = transform lns filter
