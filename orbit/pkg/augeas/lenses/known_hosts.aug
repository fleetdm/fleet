(*
Module: Known_Hosts
  Parses SSH known_hosts files

Author: Raphaël Pinson <raphink@gmail.com>

About: Reference
  This lens manages OpenSSH's known_hosts files. See `man 8 sshd` for reference.

About: License
  This file is licenced under the LGPL v2+, like the rest of Augeas.

About: Lens Usage
  Sample usage of this lens in augtool:

    * Get a key by name from ssh_known_hosts
      > print /files/etc/ssh_known_hosts/*[.="foo.example.com"]
      ...

    * Change a host's key
      > set /files/etc/ssh_known_hosts/*[.="foo.example.com"]/key "newkey"

About: Configuration files
  This lens applies to SSH known_hosts files. See <filter>.

*)

module Known_Hosts =

autoload xfm


(* View: marker
  The marker is optional, but if it is present then it must be one of
  “@cert-authority”, to indicate that the line contains a certification
  authority (CA) key, or “@revoked”, to indicate that the key contained
  on the line is revoked and must not ever be accepted.
  Only one marker should be used on a key line.
*)
let marker = [ key /@(revoked|cert-authority)/ . Sep.space ]


(* View: type
  Bits, exponent, and modulus are taken directly from the RSA host key;
  they can be obtained, for example, from /etc/ssh/ssh_host_key.pub.
  The optional comment field continues to the end of the line, and is not used.
*)
let type = [ label "type" . store Rx.neg1 ]


(* View: entry
     A known_hosts entry *)
let entry =
     let alias = [ label "alias" . store Rx.neg1 ]
  in let key = [ label "key" . store Rx.neg1 ]
  in [ Util.indent . seq "entry" . marker?
     . store Rx.neg1
     . (Sep.comma . Build.opt_list alias Sep.comma)?
     . Sep.space . type . Sep.space . key
     . Util.comment_or_eol ]

(* View: lns
     The known_hosts lens *)
let lns = (Util.empty | Util.comment | entry)*

(* Variable: filter *)
let filter = incl "/etc/ssh/ssh_known_hosts"
           . incl (Sys.getenv("HOME") . "/.ssh/known_hosts")

let xfm = transform lns filter
