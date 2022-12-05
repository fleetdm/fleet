(* Generic lens for shell-script config files like the ones found *)
(* in /etc/sysconfig, where a string needs to be split into       *)
(* single words.                                                  *)
module Shellvars_list =
  autoload xfm

  let eol = Util.eol

  let key_re = /[A-Za-z0-9_]+/
  let eq      = Util.del_str "="
  let comment = Util.comment
  let comment_or_eol = Util.comment_or_eol
  let empty   = Util.empty
  let indent  = Util.indent

  let sqword = /[^ '\t\n]+/
  let dqword = /([^ "\\\t\n]|\\\\.)+/
  let uqword = /([^ `"'\\\t\n]|\\\\.)+/
  let bqword = /`[^`\n]*`/
  let space_or_nl = /[ \t\n]+/
  let space_or_cl = space_or_nl | Rx.cl

  (* lists values of the form ...  val1 val2 val3  ... *)
  let list (word:regexp) (sep:regexp) =
    let list_value = store word in
      indent .
      [ label "value" . list_value ] .
      [ del sep " "  . label "value" . list_value ]* . indent


  (* handle single quoted lists *)
  let squote_arr = [ label "quote" . store /'/ ]
                   . (list sqword space_or_nl)? . del /'/ "'"

  (* similarly handle double quoted lists *)
  let dquote_arr = [ label "quote" . store /"/ ]
                   . (list dqword space_or_cl)? . del /"/ "\""

  (* handle unquoted single value *)
  let unquot_val = [ label "quote" . store "" ]
                 . [ label "value"  . store (uqword+ | bqword)]?


  (* lens for key value pairs *)
  let kv = [ key key_re . eq .
             ( (squote_arr | dquote_arr) . comment_or_eol
             | unquot_val . eol )
           ]

  let lns = ( comment | empty | kv )*

  let filter = incl "/etc/sysconfig/bootloader"
             . incl "/etc/sysconfig/kernel"

  let xfm = transform lns filter

(* Local Variables: *)
(* mode: caml       *)
(* End:             *)
