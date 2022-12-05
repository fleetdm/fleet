(* Parsing /etc/inittab *)
module Inittab =
   autoload xfm

   let sep = Util.del_str ":"
   let eol = Util.del_str "\n"

   let id = /[^\/#:\n]{1,4}/
   let value = /[^#:\n]*/

   let comment = Util.comment|Util.empty

   let record =
     let field (name:string) = [ label name . store value ] in
     let process = [ label "process" . store /[^#\n]*/ ] in
     let eolcomment =
       [ label "#comment" . del /#[ \t]*/ "# "
           . store /([^ \t\n].*[^ \t\n]|[^ \t\n]?)/ ] in
       [ key id . sep .
           field "runlevels" . sep .
           field "action" . sep .
           process . eolcomment? . eol ]

   let lns = ( comment | record ) *

   let xfm = transform lns (incl "/etc/inittab")


(* Local Variables: *)
(* mode: caml       *)
(* End:             *)
