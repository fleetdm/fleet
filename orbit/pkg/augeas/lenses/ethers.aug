(* Parsing /etc/ethers *)

module Ethers =
  autoload xfm

  let sep_tab = Util.del_ws_tab

  let eol = del /[ \t]*\n/ "\n"
  let indent = del /[ \t]*/ ""

  let comment = Util.comment
  let empty   = [ del /[ \t]*#?[ \t]*\n/ "\n" ]

  let word = /[^# \n\t]+/
  let address =
    let hex = /[0-9a-fA-F][0-9a-fA-F]?/ in
      hex . ":" . hex . ":" . hex . ":" . hex . ":" . hex . ":" . hex

  let record = [ seq "ether" . indent .
                              [ label "mac" . store  address ] . sep_tab .
                              [ label "ip" . store word ] . eol ]

  let lns = ( empty | comment | record ) *

  let xfm = transform lns (incl "/etc/ethers")
