(* Lens for the textual representation of Windows registry files, as used *)
(* by wine etc.                                                           *)
(* This is pretty quick and dirty, as it doesn't put a lot of finesse on  *)
(* splitting up values that have structure, e.g. hex arrays or            *)
(* collections of paths.                                                  *)
module Wine =

(* We handle Unix and DOS line endings, though we can only add one or the *)
(* other to new lines. Maybe provide a function to gather that from the   *)
(* current file ?                                                         *)
let eol = del /[ \t]*\r?\n/ "\n"
let comment = [ label "#comment" . del /[ \t]*;;[ \t]*/ ";; "
		         . store /([^ \t\r\n].*[^ \t\r\n]|[^ \t\r\n])/ . eol ]
let empty = [ eol ]
let dels = Util.del_str
let del_ws = Util.del_ws_spc

let header =
  [ label "registry" . store /[a-zA-Z0-9 \t]*[a-zA-Z0-9]/ ] .
    del /[ \t]*Version[ \t]*/ " Version " .
  [ label "version" . store /[0-9.]+/ ] . eol

let qstr =
  let re = /([^"\n]|\\\\.)*/ - /@|"@"/ in    (* " Relax, emacs *)
    dels "\"" . store re . dels "\""

let typed_val =
  ([ label "type" . store /dword|hex(\\([0-9]+\\))?/ ] . dels ":" .
    [ label "value" . store /[a-zA-Z0-9,()]+(\\\\\r?\n[ \t]*[a-zA-Z0-9,]+)*/])
  |([ label "type" . store /str\\([0-9]+\\)/ ] . dels ":" .
      dels "\"" . [ label "value" . store /[^"\n]*/ ] . dels "\"")   (* " Relax, emacs *)

let entry =
  let qkey = [ label "key" . qstr ] in
  let eq = del /[ \t]*=[ \t]*/ "=" in
  let qstore = [ label "value" . qstr ] in
  [ label "entry" . qkey . eq . (qstore|typed_val) . eol ]
  |[label "anon" . del /"?@"?/ "@" . eq . (qstore|typed_val) .eol ]

let section =
  let ts = [ label "timestamp" . store Rx.integer ] in
  [ label "section" . del /[ \t]*\\[/ "[" .
    store /[^]\n]+/ . dels "]" . (del_ws . ts)? . eol .
    (entry|empty|comment)* ]

let lns = header . (empty|comment)* . section*
