(* Parsing /boot/grub/grubenv *)

module GrubEnv =
  autoload xfm

  let eol = Util.del_str "\n"

  let comment = Util.comment
  let eq = Util.del_str "="
  let value   = /[^\\\n]*(\\\\(\\\\|\n)[^\\\n]*)*/

  let word = /[A-Za-z_][A-Za-z0-9_]*/
  let record = [ seq "target" .
                 [ label "name" . store word ] . eq .
                 [ label "value" . store value ] . eol ]

  let lns = ( comment | record ) *

  let xfm = transform lns (incl "/boot/grub/grubenv" . incl "/boot/grub2/grubenv")
