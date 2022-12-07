(*
Module: Sep
   Generic separators to build lenses

Author: Raphael Pinson <raphink@gmail.com>

About: License
  This file is licensed under the LGPL v2+, like the rest of Augeas.
*)


module Sep =

(* Variable: colon *)
let colon = Util.del_str ":"

(* Variable: semicolon *)
let semicolon = Util.del_str ";"

(* Variable: comma *)
let comma = Util.del_str ","

(* Variable: equal *)
let equal = Util.del_str "="

(* Variable: space_equal *)
let space_equal = Util.delim "="

(* Variable: space
   Deletes a <Rx.space> and default to a single space *)
let space = del Rx.space " "

(* Variable: tab
   Deletes a <Rx.space> and default to a tab *)
let tab   = del Rx.space "\t"

(* Variable: opt_space
   Deletes a <Rx.opt_space> and default to an empty string *)
let opt_space = del Rx.opt_space ""

(* Variable: opt_tab
   Deletes a <Rx.opt_space> and default to a tab *)
let opt_tab   = del Rx.opt_space "\t"

(* Variable: cl_or_space
   Deletes a <Rx.cl_or_space> and default to a single space *)
let cl_or_space = del Rx.cl_or_space " "

(* Variable: cl_or_opt_space
   Deletes a <Rx.cl_or_opt_space> and default to a single space *)
let cl_or_opt_space = del Rx.cl_or_opt_space " "

(* Variable: lbracket *)
let lbracket = Util.del_str "("

(* Variable: rbracket *)
let rbracket = Util.del_str ")"
