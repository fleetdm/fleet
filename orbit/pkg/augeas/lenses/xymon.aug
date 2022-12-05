(*
Xymon configuration
By Jason Kincl - 2012
*)

module Xymon =
  autoload xfm

let empty = Util.empty
let eol = Util.eol
let word = Rx.word
let space = Rx.space
let ip = Rx.ip 
let del_ws_spc = Util.del_ws_spc
let value_to_eol = store /[^ \t][^\n]+/
let eol_no_spc = Util.del_str "\n"

let comment = Util.comment_generic /[ \t]*[;#][ \t]*/ "# "
let include = [ key /include|dispinclude|netinclude|directory/ . del_ws_spc . value_to_eol . eol_no_spc ]
let title = [ key "title" . del_ws_spc . value_to_eol . eol_no_spc ]

(* Define host *)
let tag = del_ws_spc . [ label "tag" . store /[^ \n\t]+/ ] 
let host_ip = [ label "ip" . store ip ]
let host_hostname =  [ label "fqdn" . store word ]
let host_colon = del /[ \t]*#/ " #"
let host = [ label "host" . host_ip . del space " " . host_hostname . host_colon . tag* . eol ] 

(* Define group-compress and group-only *)
let group_extra = del_ws_spc . value_to_eol . eol_no_spc . (comment | empty | host | title)*
let group = [ key "group" . group_extra ]
let group_compress = [ key "group-compress" . group_extra ]
let group_sorted = [ key "group-sorted" . group_extra ]

let group_only_col = [ label "col" . store Rx.word ]
let group_only_cols = del_ws_spc . group_only_col . ( Util.del_str "|" . group_only_col )*
let group_only = [ key "group-only" . group_only_cols . group_extra ]

(* Have to use namespacing because page's title overlaps plain title tag *)
let page_name = store word
let page_title = [ label "pagetitle" . del_ws_spc . value_to_eol . eol_no_spc ]
let page_extra = del_ws_spc . page_name . (page_title | eol_no_spc) . (comment | empty | title | include | host)* 
                                                                     . (group | group_compress | group_sorted | group_only)*
let page = [ key /page|subpage/ . page_extra ]

let subparent_parent = [ label "parent" . store word ]
let subparent = [ key "subparent" . del_ws_spc . subparent_parent . page_extra ]

let ospage = [ key "ospage" . del_ws_spc . store word . del_ws_spc . [ label "ospagetitle" . value_to_eol . eol_no_spc ] ]

let lns = (empty | comment | include | host | title | ospage )* . (group | group_compress | group_sorted | group_only)* . (page | subparent)*

let filter = incl "/etc/xymon/hosts.cfg" . incl "/etc/xymon/pages.cfg"

let xfm = transform lns filter

