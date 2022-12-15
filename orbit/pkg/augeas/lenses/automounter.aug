(*
Module: Automounter
  Parses automounter file based maps

Author: Dominic Cleal <dcleal@redhat.com>

About: Reference
  See autofs(5)

About: License
   This file is licenced under the LGPL v2+, like the rest of Augeas.

About: Lens Usage
   To be documented

About: Configuration files
   This lens applies to /etc/auto.*, auto_*, excluding known scripts.

About: Examples
   The <Test_Automounter> file contains various examples and tests.
*)

module Automounter =
autoload xfm

(************************************************************************
 * Group:                 USEFUL PRIMITIVES
 *************************************************************************)

(* View: eol *)
let eol = Util.eol

(* View: empty *)
let empty   = Util.empty

(* View: comment *)
let comment = Util.comment

(* View: path *)
let path = /[^-+#: \t\n][^#: \t\n]*/

(* View: hostname *)
let hostname = /[^-:#\(\), \n\t][^:#\(\), \n\t]*/

(* An option label can't contain comma, comment, equals, or space *)
let optlabel = /[^,#:\(\)= \n\t]+/
let spec    = /[^,#:\(\)= \n\t][^ \n\t]*/

(* View: weight *)
let weight = Rx.integer

(* View: map_name *)
let map_name = /[^: \t\n]+/

(* View: entry_multimount_sep
   Separator for multimount entries, permits line spanning with "\" *)
let entry_multimount_sep = del /[ \t]+(\\\\[ \t]*\n[ \t]+)?/ " "

(************************************************************************
 * Group:                 ENTRIES
 *************************************************************************)

(* View: entry_key
   Key for a map entry *)
let entry_mkey = store path

(* View: entry_path
   Path component of an entry location *)
let entry_path = [ label "path" . store path ]

(* View: entry_host
   Host component with optional weight of an entry location *)
let entry_host = [ label "host" . store hostname
                   . ( Util.del_str "(" . [ label "weight"
                       . store weight ] . Util.del_str ")" )? ]

(* View: comma_sep_list
   Parses options for filesystems *)
let comma_sep_list (l:string) =
  let value = [ label "value" . Util.del_str "=" . store Rx.neg1 ] in
    let lns = [ label l . store optlabel . value? ] in
       Build.opt_list lns Sep.comma

(* View: entry_options *)
let entry_options = Util.del_str "-" . comma_sep_list "opt" . Util.del_ws_tab

(* View: entry_location
   A single location with one or more hosts, and one path *)
let entry_location = ( entry_host . ( Sep.comma . entry_host )* )?
                       . Sep.colon . entry_path

(* View: entry_locations 
   Multiple locations (each with one or more hosts), separated by spaces *)
let entry_locations = [ label "location" . counter "location"
                        . [ seq "location" . entry_location ]
                        . ( [ Util.del_ws_spc . seq "location" . entry_location ] )* ]

(* View: entry_multimount
   Parses one of many mountpoints given for a multimount line *)
let entry_multimount = entry_mkey . Util.del_ws_tab . entry_options? . entry_locations

(* View: entry_multimounts
   Parses multiple mountpoints given on an entry line *)
let entry_multimounts = [ label "mount" . counter "mount"
                          . [ seq "mount" . entry_multimount ]
                          . ( [ entry_multimount_sep . seq "mount" . entry_multimount ] )* ]

(* View: entry
   A single map entry from start to finish, including multi-mounts *)
let entry = [ seq "entry" . entry_mkey . Util.del_ws_tab . entry_options?
              . ( entry_locations | entry_multimounts ) . Util.eol ]

(* View: include
   An include line starting with a "+" and a map name *)
let include = [ seq "entry" . store "+" . Util.del_opt_ws ""
                . [ label "map" . store map_name ] . Util.eol ]

(* View: lns *)
let lns = ( empty | comment | entry | include ) *

(* Variable: filter
   Exclude scripts/executable maps from here *)
let filter = incl "/etc/auto.*"
           . incl "/etc/auto_*"
           . excl "/etc/auto.master"
           . excl "/etc/auto_master"
           . excl "/etc/auto.net"
           . excl "/etc/auto.smb"
           . Util.stdexcl

let xfm = transform lns filter
