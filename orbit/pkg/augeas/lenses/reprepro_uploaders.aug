(*
Module: Reprepro_Uploaders
  Parses reprepro's uploaders files

Author: Raphael Pinson <raphink@gmail.com>

About: Reference
  This lens tries to keep as close as possible to `man 1 reprepro` where possible.

About: License
   This file is licenced under the LGPL v2+, like the rest of Augeas.
About: Lens Usage
   See <lns>.

About: Configuration files
   This lens applies to reprepro's uploaders files.

About: Examples
   The <Test_Reprepro_Uploaders> file contains various examples and tests.
*)

module Reprepro_Uploaders =

(* View: logic_construct_condition
   A logical construction for <condition> and <condition_list> *)
let logic_construct_condition (kw:string) (lns:lens) =
    [ label kw . lns ]
  . [ Sep.space . key kw . Sep.space . lns ]*

(* View: logic_construct_field
   A generic definition for <condition_field> *)
let logic_construct_field (kw:string) (sep:string) (lns:lens) =
    [ label kw . lns ]
  . [ Build.xchgs sep kw . lns ]*

(* View: condition_re
   A condition can be of several types:

   - source
   - byhand
   - sections
   - sections contain
   - binaries
   - binaries contain
   - architectures
   - architectures contain

   While the lens technically also accepts "source contain"
   and "byhand contain", these are not understood by reprepro.

   The "contain" types are built by adding a "contain" subnode.
   See the <condition_field> definition.

 *)
let condition_re =
    "source"
  | "byhand"
  | "sections"
  | "binaries"
  | "architectures"
  | "distribution"

(* View: condition_field
   A single condition field is an 'or' node.
   It may contain several values, listed in 'or' subnodes:

   > $reprepro/allow[1]/and/or = "architectures"
   > $reprepro/allow[1]/and/or/or[1] = "i386"
   > $reprepro/allow[1]/and/or/or[2] = "amd64"
   > $reprepro/allow[1]/and/or/or[3] = "all"

 *)
let condition_field =
  let sto_condition = Util.del_str "'" . store /[^'\n]+/ . Util.del_str "'" in
    [ key "not" . Sep.space ]? .
    store condition_re
  . [ Sep.space . key "contain" ]?
  . Sep.space
  . logic_construct_field "or" "|" sto_condition

(* View: condition
   A condition is an 'and' node,
   representing a union of <condition_fields>,
   listed under 'or' subnodes:

   > $reprepro/allow[1]/and
   > $reprepro/allow[1]/and/or = "architectures"
   > $reprepro/allow[1]/and/or/or[1] = "i386"
   > $reprepro/allow[1]/and/or/or[2] = "amd64"
   > $reprepro/allow[1]/and/or/or[3] = "all"

 *)
let condition =
    logic_construct_condition "or" condition_field

(* View: condition_list
   A list of <conditions>, inspired by Debctrl.dependency_list
   An upload condition list is either the wildcard '*', stored verbatim,
   or an intersection of conditions listed under 'and' subnodes:

   > $reprepro/allow[1]/and[1]
   > $reprepro/allow[1]/and[1]/or = "architectures"
   > $reprepro/allow[1]/and[1]/or/or[1] = "i386"
   > $reprepro/allow[1]/and[1]/or/or[2] = "amd64"
   > $reprepro/allow[1]/and[1]/or/or[3] = "all"
   > $reprepro/allow[1]/and[2]
   > $reprepro/allow[1]/and[2]/or = "sections"
   > $reprepro/allow[1]/and[2]/or/contain
   > $reprepro/allow[1]/and[2]/or/or = "main"

 *)
let condition_list =
    store "*"
  | logic_construct_condition "and" condition

(* View: by_key
   When a key is used to authenticate packages,
   the value can either be a key ID or "any":

   > $reprepro/allow[1]/by/key = "ABCD1234"
   > $reprepro/allow[2]/by/key = "any"

 *)
let by_key =
  let any_key   = [ store "any" . Sep.space
                  . key "key" ] in
  let named_key = [ key "key" . Sep.space
                  . store (Rx.word - "any") ] in
    value "key" . (any_key | named_key)

(* View: by_group
   Authenticate packages by a groupname.

   > $reprepro/allow[1]/by/group = "groupname"

 *)
let by_group = value "group"
             . [ key "group" . Sep.space
             . store Rx.word ]

(* View: by
   <by> statements define who is allowed to upload.
   It can be simple keywords, like "anybody" or "unsigned",
   or a key ID, in which case a "key" subnode is added:

   > $reprepro/allow[1]/by/key = "ABCD1234"
   > $reprepro/allow[2]/by/key = "any"
   > $reprepro/allow[3]/by = "anybody"
   > $reprepro/allow[4]/by = "unsigned"

 *)
let by =
    [ key "by" . Sep.space
         . ( store ("anybody"|"unsigned")
           | by_key | by_group ) ]

(* View: allow
   An allow entry, e.g.:

   > $reprepro/allow[1]
   > $reprepro/allow[1]/and[1]
   > $reprepro/allow[1]/and[1]/or = "architectures"
   > $reprepro/allow[1]/and[1]/or/or[1] = "i386"
   > $reprepro/allow[1]/and[1]/or/or[2] = "amd64"
   > $reprepro/allow[1]/and[1]/or/or[3] = "all"
   > $reprepro/allow[1]/and[2]
   > $reprepro/allow[1]/and[2]/or = "sections"
   > $reprepro/allow[1]/and[2]/or/contain
   > $reprepro/allow[1]/and[2]/or/or = "main"
   > $reprepro/allow[1]/by = "key"
   > $reprepro/allow[1]/by/key = "ABCD1234"

 *)
let allow =
    [ key "allow" . Sep.space
  . condition_list . Sep.space
  . by . Util.eol ]

(* View: group
   A group declaration *)
let group =
     let add = [ key "add" . Sep.space
             . store Rx.word ]
  in let contains = [ key "contains" . Sep.space
                    . store Rx.word ]
  in let empty = [ key "empty" ]
  in let unused = [ key "unused" ]
  in [ key "group" . Sep.space
     . store Rx.word . Sep.space
     . (add | contains | empty | unused) . Util.eol ]

(* View: entry
   An entry is either an <allow> statement
   or a <group> definition.
 *)
let entry = allow | group

(* View: lns
   The lens is made of <Util.empty>, <Util.comment> and <entry> lines *)
let lns = (Util.empty|Util.comment|entry)*
