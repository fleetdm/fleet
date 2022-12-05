(* Parsing grub's device.map *)

module Device_map =
  autoload xfm

  let sep_tab = Sep.tab
  let eol     = Util.eol
  let fspath  = Rx.fspath
  let del_str = Util.del_str

  let comment = Util.comment
  let empty   = Util.empty

  let dev_name = /(h|f|c)d[0-9]+(,[0-9a-zA-Z]+){0,2}/
  let dev_hex  = Rx.hex
  let dev_dec  = /[0-9]+/

  let device = del_str "(" . key ( dev_name | dev_hex | dev_dec ) .  del_str ")"

  let map = [ device . sep_tab . store fspath . eol ]

  let lns = ( empty | comment | map ) *

  let xfm = transform lns (incl "/boot/*/device.map")

(* Local Variables: *)
(* mode: caml *)
(* End: *)
