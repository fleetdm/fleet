(*
Module: Iscsid
Parses iscsid configuration file
Author: Joey Boggs <jboggs@redhat.com>
About: Reference
This lens is targeted at /etc/iscsi/iscsid.conf
*)
module Iscsid =
  autoload xfm

  let filter = incl "/etc/iscsi/iscsid.conf"

  let eol = Util.eol
  let indent = Util.indent
  let key_re = /[][A-Za-z0-9_.-]+/
  let eq = del /[ \t]*=[ \t]*/ " = "
  let value_re = /[^ \t\n](.*[^ \t\n])?/

  let comment = [ indent . label "#comment" . del /[#;][ \t]*/ "# "
        . store /([^ \t\n].*[^ \t\n]|[^ \t\n])/ . eol ]

  let empty = Util.empty

  let kv = [ indent . key key_re . eq . store value_re . eol ]

  let lns = (empty | comment | kv) *

  let xfm = transform lns filter
