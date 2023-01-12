(*
Module: Approx
  Parses /etc/approx/approx.conf

Author: David Lutterkort <lutter@redhat.com>

About: Reference
  This lens tries to keep as close as possible to `man 5 approx.conf` where possible.

About: License
   This file is licenced under the LGPL v2+, like the rest of Augeas.

About: Lens Usage
   See <lns>.

About: Configuration files
   This lens applies to /etc/approx/approx.conf.

About: Examples
   The <Test_Approx> file contains various examples and tests.
*)

module Approx =
  autoload xfm

  (* Variable: eol
     An <Util.eol> *)
  let eol = Util.eol

  (* Variable: indent
     An <Util.indent> *)
  let indent = Util.indent

  (* Variable: key_re *)
  let key_re = /\$?[A-Za-z0-9_.-]+/

  (* Variable: sep *)
  let sep = /[ \t]+/

  (* Variable: value_re *)
  let value_re = /[^ \t\n](.*[^ \t\n])?/

  (* View: comment *)
  let comment = [ indent . label "#comment" . del /[#;][ \t]*/ "# "
        . store /([^ \t\n].*[^ \t\n]|[^ \t\n])/ . eol ]

  (* View: empty
     An <Util.empty> *)
  let empty = Util.empty

  (* View: kv *)
  let kv = [ indent . key key_re . del sep " " . store value_re . eol ]

  (* View: lns *)
  let lns = (empty | comment | kv) *

  (* View: filter *)
  let filter = incl "/etc/approx/approx.conf"
  let xfm = transform lns filter
