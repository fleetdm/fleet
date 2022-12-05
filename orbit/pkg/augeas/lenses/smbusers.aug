(*
Module: SmbUsers
  Parses Samba username maps

Author: Raphael Pinson <raphink@gmail.com>

About: License
   This file is licenced under the LGPL v2+, like the rest of Augeas.

About: Lens Usage
   To be documented

About: Examples
   The <Test_SmbUsers> file contains various examples and tests.
*)

module SmbUsers =

autoload xfm

(* View: entry *)
let entry =
     let username = [ label "username" . store Rx.no_spaces ]
  in let usernames = Build.opt_list username Sep.space
  in Build.key_value_line Rx.word Sep.space_equal usernames

(* View: lns *)
let lns = (Util.empty | (Util.comment_generic /[ \t]*[#;][ \t]*/ "# ") | entry)*

(* Variable: filter *)
let filter = incl "/etc/samba/smbusers"
           . incl "/etc/samba/usermap.txt"

let xfm = transform lns filter
