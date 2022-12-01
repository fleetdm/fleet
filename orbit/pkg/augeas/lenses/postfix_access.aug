(* Parsing /etc/postfix/access *)

module Postfix_Access =
  autoload xfm

  let sep_tab = Util.del_ws_tab
  let sep_spc = Util.del_ws_spc

  let eol = del /[ \t]*\n/ "\n"
  let indent = del /[ \t]*/ ""

  let comment = Util.comment
  let empty   = Util.empty

  let char = /[^# \n\t]/
  let text =
    let cont = /\n[ \t]+/ in
    let any = /[^#\n]/ in
    char | (char . (any | cont)* .char)

  let word = char+
  let record = [ seq "spec" .
                  [ label "pattern" . store  word ] . sep_tab .
                  [ label "action" . store word ] .
                  [ label "parameters" . sep_spc . store text ]? . eol ]

  let lns = ( empty | comment | record )*

  let xfm = transform lns (incl "/etc/postfix/access" . incl "/usr/local/etc/postfix/access")
