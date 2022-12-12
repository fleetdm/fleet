(* Variation of the Shellvars lens                                     *)
(* Supports only what's needed to handle sysconfig files               *)
(* Modified to strip quotes. In the put direction, add double quotes   *)
(* around values that need them                                        *)
(* To keep things simple, we also do not support shell variable arrays *)
module Sysconfig =

  let eol = Shellvars.eol
  let semicol_eol = Shellvars.semicol_eol

  let key_re = Shellvars.key_re
  let eq = Util.del_str "="

  let eol_for_comment = del /([ \t]*\n)([ \t]*(#[ \t]*)?\n)*/ "\n"
  let comment = Util.comment_generic_seteol /[ \t]*#[ \t]*/ "# " eol_for_comment
  let comment_or_eol = Shellvars.comment_or_eol

  let empty   = Util.empty

  let bchar = /[^; \t\n"'\\]|\\\\./ (* " Emacs, relax *)
  let qchar = /["']/  (* " Emacs, relax *)

  (* We split the handling of right hand sides into a few cases:
   *   bare  - strings that contain no spaces, optionally enclosed in
   *           single or double quotes
   *   quot  - strings that must be enclosed in single or double quotes
   *   dquot - strings that contain at least one space or apostrophe,
   *           which must be enclosed in double quotes
   *   squot - strings that contain an unescaped double quote
   *)
  let bare = Quote.do_quote_opt (store bchar+)

  let quot =
    let word = bchar* . /[; \t]/ . bchar* in
    Quote.do_quote (store word+)

  let dquot =
    let char = /[^"\\]|\\\\./ in             (* " *)
    let word = char* . "'" . char* in
    Quote.do_dquote (store word+)

  let squot =
    (* We do not allow escaped double quotes in single quoted strings, as  *)
    (* that leads to a put ambiguity with bare, e.g. for the string '\"'.  *)
    let char = /[^'\\]|\\\\[^"]/ in           (* " *)
    let word = char* . "\"" . char* in
    Quote.do_squote (store word+)

  let kv (value:lens) =
    let export = Shellvars.export in
    let indent = Util.del_opt_ws "" in
    [ indent . export? . key key_re . eq . value . comment_or_eol ]

  let assign =
    let nothing = del /(""|'')?/ "" . value "" in
    kv nothing | kv bare | kv quot | kv dquot | kv squot

  let var_action = Shellvars.var_action

  let unset = [ var_action "unset" . comment_or_eol ]
  let bare_export = [ var_action "export" . comment_or_eol ]

  let source = [ Shellvars.source . comment_or_eol ]

  let lns = empty* . (comment | source | assign | unset | bare_export)*

(*
  Examples:

  abc   -> abc -> abc
  "abc" -> abc -> abc
  "a b" -> a b -> "a b"
  'a"b' -> a"b -> 'a"b'
*)
