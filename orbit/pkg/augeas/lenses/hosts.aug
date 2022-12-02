(* Parsing /etc/hosts *)

module Hosts =
  autoload xfm

  let word = /[^# \n\t]+/
  let record = [ seq "host" . Util.indent .
                              [ label "ipaddr" . store  word ] . Sep.tab .
                              [ label "canonical" . store word ] .
                              [ label "alias" . Sep.space . store word ]*
                 . Util.comment_or_eol ]

  let lns = ( Util.empty | Util.comment | record ) *

  let xfm = transform lns (incl "/etc/hosts")
