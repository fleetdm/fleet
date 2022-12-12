(*
Module: Tmpfiles
  Parses systemd tmpfiles.d files

Author: Julien Pivotto <roidelapluie@inuits.eu>

About: Reference
  This lens tries to keep as close as possible to `man 5 tmpfiles.d` where possible.

About: License
   This file is licenced under the LGPL v2+, like the rest of Augeas.

About: Lens Usage
   To be documented

About: Configuration files
   This lens applies to /etc/tmpfiles.d/*.conf /usr/lib/tmpfiles.d/*.conf and
   /run/tmpfiles.d/*.conf. See <filter>.

About: Examples
   The <Test_Tmpfiles> file contains various examples and tests.
*)

module Tmpfiles =
  autoload xfm

(************************************************************************
 * Group:                 USEFUL PRIMITIVES
 *************************************************************************)

(* Group: Comments and empty lines *)

  (* View: sep_spc
Space *)
  let sep_spc = Sep.space

  (* View: sep_opt_spc
Optional space (for the beginning of the lines) *)
  let sep_opt_spc = Sep.opt_space

  (* View: comment
Comments *)
  let comment = Util.comment

  (* View: empty
Empty lines *)
  let empty   = Util.empty

(* Group: Lense-specific primitives *)

  (* View: type
One letter. Some of them can have a "+" and all can have an
exclamation mark ("!") and/or minus sign ("-").

Not all letters are valid.
*)
  let type     = /([fFwdDevqQpLcbCxXrRzZtThHaAm]|[fFwpLcbaA]\+)!?-?/

  (* View: mode
"-", or 3-4 bytes. Optionally starts with a "~". *)
  let mode     = /(-|~?[0-7]{3,4})/

  (* View: age
"-", or one of the formats seen in the manpage: 10d, 5seconds, 1y5days.
optionally starts with a "~'. *)
  let age      = /(-|(~?[0-9]+(s|m|min|h|d|w|ms|us|((second|minute|hour|day|week|millisecond|microsecond)s?))?)+)/

  (* View: argument
The last field. It can contain spaces. *)
  let argument = /([^# \t\n][^#\n]*[^# \t\n]|[^# \t\n])/

  (* View: field
Applies to the other fields: path, gid and uid fields *)
  let field    = /[^# \t\n]+/

  (* View: record
A valid record, one line in the file.
Only the two first fields are mandatory. *)
  let record = [ seq "record" . sep_opt_spc .
                   [ label "type" . store type ] . sep_spc .
                   [ label "path" . store field ] . ( sep_spc .
                   [ label "mode" . store mode ] . ( sep_spc .
                   [ label "uid" . store field ] . ( sep_spc .
                   [ label "gid" . store field ] . ( sep_spc .
                   [ label "age" . store age ] . ( sep_spc .
                   [ label "argument" . store argument ] )? )? )? )? )? .
                     Util.comment_or_eol ]

(************************************************************************
 * Group:                 THE TMPFILES LENSE
 *************************************************************************)

  (* View: lns
The tmpfiles lens.
Each line can be a comment, a record or empty. *)
  let lns = ( empty | comment | record ) *

  (* View: filter *)
  let filter = incl "/etc/tmpfiles.d/*.conf"
             . incl "/usr/lib/tmpfiles.d/*.conf"
             . incl "/run/tmpfiles.d/*.conf"

  let xfm = transform lns filter

(* Local Variables: *)
(* mode: caml *)
(* End: *)
