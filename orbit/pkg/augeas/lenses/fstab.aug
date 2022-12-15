(* Parsing /etc/fstab *)

module Fstab =
  autoload xfm

  let sep_tab = Sep.tab
  let sep_spc = Sep.space
  let comma   = Sep.comma
  let eol     = Util.eol

  let comment = Util.comment
  let empty   = Util.empty

  let file    = /[^# \t\n]+/

  (* An option label can't contain comma, comment, equals, or space *)
  let optlabel = /[^,#= \n\t]+/
  let spec    = /[^,# \n\t][^ \n\t]*/

  let comma_sep_list (l:string) =
    let value = [ label "value" . Util.del_str "=" . ( store Rx.neg1 )? ] in
      let lns = [ label l . store optlabel . value? ] in
         Build.opt_list lns comma

  let record = [ seq "mntent" .
                   Util.indent .
                   [ label "spec" . store spec ] . sep_tab .
                   [ label "file" . store file ] . sep_tab .
                   comma_sep_list "vfstype" .
                   (sep_tab . comma_sep_list "opt" .
                    (sep_tab . [ label "dump" . store /[0-9]+/ ] .
                     ( sep_spc . [ label "passno" . store /[0-9]+/ ])? )? )?
                 . Util.comment_or_eol ]

  let lns = ( empty | comment | record ) *
  let filter = incl "/etc/fstab"
             . incl "/etc/mtab"

  let xfm = transform lns filter

(* Local Variables: *)
(* mode: caml *)
(* End: *)
