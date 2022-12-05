(* Parses entries in /etc/securetty

   Author: Simon Josi <josi@yokto.net> 
*)
module Securetty =
   autoload xfm

   let word = /[^ \t\n#]+/
   let eol = Util.eol
   let empty = Util.empty
   let comment = Util.comment
   let comment_or_eol = Util.comment_or_eol

   let record = [ seq "securetty" . store word . comment_or_eol ]
   let lns = ( empty | comment | record )*

   let filter = (incl "/etc/securetty")
   let xfm = transform lns filter
