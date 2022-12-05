(*
Module: Quote
  Generic module providing useful primitives for quoting

Author: Raphael Pinson <raphael.pinson@camptocamp.com>

About: License
   This file is licenced under the LGPL v2+, like the rest of Augeas.

About: Lens Usage
   This is a generic module which doesn't apply to files directly.
   You can use its definitions to build lenses that require quoted values.
   It provides several levels of definitions, allowing to define more or less fine-grained quoted values:

     - the quote separators are separators that are useful to define quoted values;
     - the quoting functions are useful wrappers to easily enclose a lens in various kinds of quotes (single, double, any, optional or not);
     - the quoted values definitions are common quoted patterns. They use the quoting functions in order to provide useful shortcuts for commonly met needs. In particular, the <quote_spaces> (and similar) function force values that contain spaces to be quoted, but allow values without spaces to be unquoted.

About: Examples
   The <Test_Quote> file contains various examples and tests.
*)

module Quote =

(* Group: QUOTE SEPARATORS *)

(* Variable: dquote
     A double quote *)
let dquote = Util.del_str "\""

(* Variable: dquote_opt
     An optional double quote, default to double *)
let dquote_opt = del /"?/ "\""

(* Variable: dquote_opt_nil
     An optional double quote, default to nothing *)
let dquote_opt_nil = del /"?/ ""

(* Variable: squote
     A single quote *)
let squote = Util.del_str "'"

(* Variable: squote_opt
     An optional single quote, default to single *)
let squote_opt = del /'?/ "'"

(* Variable: squote_opt_nil
     An optional single quote, default to nothing *)
let squote_opt_nil = del /'?/ ""

(* Variable: quote
     A quote, either double or single, default to double *)
let quote = del /["']/ "\""

(* Variable: quote_opt
     An optional quote, either double or single, default to double *)
let quote_opt = del /["']?/ "\""

(* Variable: quote_opt_nil
     An optional quote, either double or single, default to nothing *)
let quote_opt_nil = del /["']?/ ""


(* Group: QUOTING FUNCTIONS *)

(*
View: do_dquote
  Enclose a lens in <dquote>s

  Parameters:
    body:lens - the lens to be enclosed
*)
let do_dquote (body:lens) =
  square dquote body dquote

(*
View: do_dquote_opt
  Enclose a lens in optional <dquote>s,
  use <dquote>s by default.

  Parameters:
    body:lens - the lens to be enclosed
*)
let do_dquote_opt (body:lens) =
  square dquote_opt body dquote_opt

(*
View: do_dquote_opt_nil
  Enclose a lens in optional <dquote>s,
  default to no quotes.

  Parameters:
    body:lens - the lens to be enclosed
*)
let do_dquote_opt_nil (body:lens) =
  square dquote_opt_nil body dquote_opt_nil

(*
View: do_squote
  Enclose a lens in <squote>s

  Parameters:
    body:lens - the lens to be enclosed
*)
let do_squote (body:lens) =
  square squote body squote

(*
View: do_squote_opt
  Enclose a lens in optional <squote>s,
  use <squote>s by default.

  Parameters:
    body:lens - the lens to be enclosed
*)
let do_squote_opt (body:lens) =
  square squote_opt body squote_opt

(*
View: do_squote_opt_nil
  Enclose a lens in optional <squote>s,
  default to no quotes.

  Parameters:
    body:lens - the lens to be enclosed
*)
let do_squote_opt_nil (body:lens) =
  square squote_opt_nil body squote_opt_nil

(*
View: do_quote
  Enclose a lens in <quote>s.

  Parameters:
    body:lens - the lens to be enclosed
*)
let do_quote (body:lens) =
  square quote body quote

(*
View: do_quote
  Enclose a lens in options <quote>s.

  Parameters:
    body:lens - the lens to be enclosed
*)
let do_quote_opt (body:lens) =
  square quote_opt body quote_opt

(*
View: do_quote
  Enclose a lens in options <quote>s,
  default to no quotes.

  Parameters:
    body:lens - the lens to be enclosed
*)
let do_quote_opt_nil (body:lens) =
  square quote_opt_nil body quote_opt_nil


(* Group: QUOTED VALUES *)

(* View: double
     A double-quoted value *)
let double =
     let body = store /[^\n]*/
  in do_dquote body

(* Variable: double_opt_re
     The regexp to store when value
     is optionally double-quoted *)
let double_opt_re = /[^\n\t "]([^\n"]*[^\n\t "])?/

(* View: double_opt
     An optionally double-quoted value
     Double quotes are not allowed in value
     Value cannot begin or end with spaces *)
let double_opt =
     let body = store double_opt_re
  in do_dquote_opt body

(* View: single
     A single-quoted value *)
let single =
     let body = store /[^\n]*/
  in do_squote body

(* Variable: single_opt_re
     The regexp to store when value
     is optionally single-quoted *)
let single_opt_re = /[^\n\t ']([^\n']*[^\n\t '])?/

(* View: single_opt
     An optionally single-quoted value
     Single quotes are not allowed in value
     Value cannot begin or end with spaces *)
let single_opt =
     let body = store single_opt_re
  in do_squote_opt body

(* View: any
     A quoted value *)
let any =
     let body = store /[^\n]*/
  in do_quote body

(* Variable: any_opt_re
     The regexp to store when value
     is optionally single- or double-quoted *)
let any_opt_re = /[^\n\t "']([^\n"']*[^\n\t "'])?/

(* View: any_opt
     An optionally quoted value
     Double or single quotes are not allowed in value
     Value cannot begin or end with spaces *)
let any_opt =
     let body = store any_opt_re
  in do_quote_opt body

(*
View: quote_spaces
  Make quotes mandatory if value contains spaces,
  and optional if value doesn't contain spaces.

Parameters:
  lns:lens - the lens to be enclosed
*)
let quote_spaces (lns:lens) =
     (* bare has no spaces, and is optionally quoted *)
     let bare = Quote.do_quote_opt (store /[^"' \t\n]+/)
     (* quoted has at least one space, and must be quoted *)
  in let quoted = Quote.do_quote (store /[^"'\n]*[ \t]+[^"'\n]*/)
  in [ lns . bare ] | [ lns . quoted ]

(*
View: dquote_spaces
  Make double quotes mandatory if value contains spaces,
  and optional if value doesn't contain spaces.

Parameters:
  lns:lens - the lens to be enclosed
*)
let dquote_spaces (lns:lens) =
     (* bare has no spaces, and is optionally quoted *)
     let bare = Quote.do_dquote_opt (store /[^" \t\n]+/)
     (* quoted has at least one space, and must be quoted *)
  in let quoted = Quote.do_dquote (store /[^"\n]*[ \t]+[^"\n]*/)
  in [ lns . bare ] | [ lns . quoted ]

(*
View: squote_spaces
  Make single quotes mandatory if value contains spaces,
  and optional if value doesn't contain spaces.

Parameters:
  lns:lens - the lens to be enclosed
*)
let squote_spaces (lns:lens) =
     (* bare has no spaces, and is optionally quoted *)
     let bare = Quote.do_squote_opt (store /[^' \t\n]+/)
     (* quoted has at least one space, and must be quoted *)
  in let quoted = Quote.do_squote (store /[^'\n]*[ \t]+[^'\n]*/)
  in [ lns . bare ] | [ lns . quoted ]
