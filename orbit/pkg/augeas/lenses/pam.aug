(*
Module: Pam
  Parses /etc/pam.conf and /etc/pam.d/* service files

Author: David Lutterkort <lutter@redhat.com>

About: Reference
  This lens tries to keep as close as possible to `man pam.conf` where
  possible.

About: Licence
  This file is licensed under the LGPL v2+, like the rest of Augeas.

About: Lens Usage

About: Configuration files
  This lens autoloads /etc/pam.d/* for service specific files. See <filter>.
  It provides a lens for /etc/pam.conf, which is used in the PamConf module.
*)
module Pam =
  autoload xfm

  let eol = Util.eol
  let indent = Util.indent
  let space = del /([ \t]|\\\\\n)+/ " "

  (* For the control syntax of [key=value ..] we could split the key value *)
  (* pairs into an array and generate a subtree control/N/KEY = VALUE      *)
  (* The valid control values if the [...] syntax is not used, is          *)
  (*   required|requisite|optional|sufficient|include|substack             *)
  (* We allow more than that because this list is not case sensitive and   *)
  (* to be more lenient with typos                                         *)
  let control = /(\[[^]#\n]*\]|[a-zA-Z]+)/
  let word = /([^# \t\n\\]|\\\\.)+/
  (* Allowed types *)
  let types = /(auth|session|account|password)/i

  (* This isn't entirely right: arguments enclosed in [ .. ] can contain  *)
  (* a ']' if escaped with a '\' and can be on multiple lines ('\')       *)
  let argument = /(\[[^]#\n]+\]|[^[#\n \t\\][^#\n \t\\]*)/

  let comment = Util.comment
  let comment_or_eol = Util.comment_or_eol
  let empty   = Util.empty


  (* Not mentioned in the man page, but Debian uses the syntax             *)
  (*   @include module                                                     *)
  (* quite a bit                                                           *)
  let include = [ indent . Util.del_str "@" . key "include" .
                  space . store word . eol ]

  (* Shared with PamConf *)
  let record = [ label "optional" . del "-" "-" ]? .
               [ label "type" . store types ] .
               space .
               [ label "control" . store control] .
               space .
               [ label "module" . store word ] .
               [ space . label "argument" . store argument ]* .
               comment_or_eol

  let record_svc = [ seq "record" . indent . record ]

  let lns = ( empty | comment | include | record_svc ) *

  let filter = incl "/etc/pam.d/*"
             . excl "/etc/pam.d/allow.pamlist"
             . excl "/etc/pam.d/README"
             . Util.stdexcl

  let xfm = transform lns filter

(* Local Variables: *)
(* mode: caml       *)
(* End:             *)
