(*
Module: Build
   Generic functions to build lenses

Author: Raphael Pinson <raphink@gmail.com>

About: License
  This file is licensed under the LGPL v2+, like the rest of Augeas.

About: Reference
  This file provides generic functions to build Augeas lenses
*)


module Build =

let eol = Util.eol

(************************************************************************
 * Group:               GENERIC CONSTRUCTIONS
 ************************************************************************)

(************************************************************************
 * View: brackets
 *   Put a lens inside brackets
 *
 *   Parameters:
 *     l:lens   - the left bracket lens
 *     r: lens  - the right bracket lens
 *     lns:lens - the lens to put inside brackets
 ************************************************************************)
let brackets (l:lens) (r:lens) (lns:lens) = l . lns . r


(************************************************************************
 * Group:             LIST CONSTRUCTIONS
 ************************************************************************)

(************************************************************************
 * View: list
 *   Build a list of identical lenses separated with a given separator
 *   (at least 2 elements)
 *
 *   Parameters:
 *     lns:lens - the lens to repeat in the list
 *     sep:lens - the separator lens, which can be taken from the <Sep> module
 ************************************************************************)
let list (lns:lens) (sep:lens) = lns . ( sep . lns )+


(************************************************************************
 * View: opt_list
 *   Same as <list>, but there might be only one element in the list
 *
 *   Parameters:
 *     lns:lens - the lens to repeat in the list
 *     sep:lens - the separator lens, which can be taken from the <Sep> module
 ************************************************************************)
let opt_list (lns:lens) (sep:lens) = lns . ( sep . lns )*


(************************************************************************
 * Group:                   LABEL OPERATIONS
 ************************************************************************)

(************************************************************************
 * View: xchg
 *   Replace a pattern with a different label in the tree,
 *   thus emulating a key but allowing to replace the keyword
 *   with a different value than matched
 *
 *   Parameters:
 *     m:regexp - the pattern to match
 *     d:string - the default value when a node in created
 *     l:string - the label to apply for such nodes
 ************************************************************************)
let xchg (m:regexp) (d:string) (l:string) = del m d . label l

(************************************************************************
 * View: xchgs
 *   Same as <xchg>, but the pattern is the default string
 *
 *   Parameters:
 *     m:string - the string to replace, also used as default
 *     l:string - the label to apply for such nodes
 ************************************************************************)
let xchgs (m:string) (l:string) = xchg m m l


(************************************************************************
 * Group:                   SUBNODE CONSTRUCTIONS
 ************************************************************************)

(************************************************************************
 * View: key_value_line
 *   A subnode with a keyword, a separator and a storing lens,
 *   and an end of line
 *
 *   Parameters:
 *     kw:regexp - the pattern to match as key
 *     sep:lens  - the separator lens, which can be taken from the <Sep> module
 *     sto:lens  - the storing lens
 ************************************************************************)
let key_value_line (kw:regexp) (sep:lens) (sto:lens) =
                                   [ key kw . sep . sto . eol ]

(************************************************************************
 * View: key_value_line_comment
 *   Same as <key_value_line>, but allows to have a comment in the end of a line
 *   and an end of line
 *
 *   Parameters:
 *     kw:regexp    - the pattern to match as key
 *     sep:lens     - the separator lens, which can be taken from the <Sep> module
 *     sto:lens     - the storing lens
 *     comment:lens - the comment lens, which can be taken from <Util>
 ************************************************************************)
let key_value_line_comment (kw:regexp) (sep:lens) (sto:lens) (comment:lens) =
                                   [ key kw . sep . sto . (eol|comment) ]

(************************************************************************
 * View: key_value
 *   Same as <key_value_line>, but does not end with an end of line
 *
 *   Parameters:
 *     kw:regexp - the pattern to match as key
 *     sep:lens  - the separator lens, which can be taken from the <Sep> module
 *     sto:lens  - the storing lens
 ************************************************************************)
let key_value (kw: regexp) (sep:lens) (sto:lens) =
                                   [ key kw . sep . sto ]

(************************************************************************
 * View: key_ws_value
 *
 *   Store a key/value pair where key and value are separated by whitespace
 *   and the value goes to the end of the line. Leading and trailing
 *   whitespace is stripped from the value. The end of line is consumed by
 *   this lens
 *
 *   Parameters:
 *     kw:regexp - the pattern to match as key
 ************************************************************************)
let key_ws_value (kw:regexp) =
  key_value_line kw Util.del_ws_spc (store Rx.space_in)

(************************************************************************
 * View: flag
 *   A simple flag subnode, consisting of a single key
 *
 *   Parameters:
 *     kw:regexp - the pattern to match as key
 ************************************************************************)
let flag (kw:regexp) = [ key kw ]

(************************************************************************
 * View: flag_line
 *   A simple flag line, consisting of a single key
 *
 *   Parameters:
 *     kw:regexp - the pattern to match as key
 ************************************************************************)
let flag_line (kw:regexp) = [ key kw . eol ]


(************************************************************************
 * Group:                   BLOCK CONSTRUCTIONS
 ************************************************************************)

(************************************************************************
 * View: block_generic
 *   A block enclosed in brackets
 *
 *   Parameters:
 *     entry:lens                - the entry to be stored inside the block.
 *                                 This entry should include <Util.empty>
 *                                 or its equivalent if necessary.
 *     entry_noindent:lens       - the entry to be stored inside the block,
 *                                 without indentation.
 *                                 This entry should not include <Util.empty>
 *     entry_noeol:lens          - the entry to be stored inside the block,
 *                                 without eol.
 *                                 This entry should not include <Util.empty>
 *     entry_noindent_noeol:lens - the entry to be stored inside the block,
 *                                 without indentation or eol.
 *                                 This entry should not include <Util.empty>
 *     comment:lens              - the comment lens used in the block
 *     comment_noindent:lens     - the comment lens used in the block,
 *                                 without indentation.
 *     ldelim_re:regexp          - regexp for the left delimiter
 *     rdelim_re:regexp          - regexp for the right delimiter
 *     ldelim_default:string     - default value for the left delimiter
 *     rdelim_default:string     - default value for the right delimiter
 ************************************************************************)
let block_generic
     (entry:lens) (entry_noindent:lens)
     (entry_noeol:lens) (entry_noindent_noeol:lens)
     (comment:lens) (comment_noindent:lens)
     (ldelim_re:regexp) (rdelim_re:regexp)
     (ldelim_default:string) (rdelim_default:string) =
     let block_single = entry_noindent_noeol | comment_noindent
  in let block_start  = entry_noindent | comment_noindent
  in let block_middle = (entry | comment)*
  in let block_end    = entry_noeol | comment
  in del ldelim_re ldelim_default
     . ( ( block_start . block_middle . block_end )
       | block_single )
     . del rdelim_re rdelim_default

(************************************************************************
 * View: block_setdefault
 *   A block enclosed in brackets
 *
 *   Parameters:
 *     entry:lens - the entry to be stored inside the block.
 *                  This entry should not include <Util.empty>,
 *                  <Util.comment> or <Util.comment_noindent>,
 *                  should not be indented or finish with an eol.
 *     ldelim_re:regexp      - regexp for the left delimiter
 *     rdelim_re:regexp      - regexp for the left delimiter
 *     ldelim_default:string - default value for the left delimiter
 *     rdelim_default:string - default value for the right delimiter
 ************************************************************************)
let block_setdelim (entry:lens)
                     (ldelim_re:regexp)
                     (rdelim_re:regexp)
                     (ldelim_default:string)
                     (rdelim_default:string) =
    block_generic (Util.empty | Util.indent . entry . eol)
                  (entry . eol) (Util.indent . entry) entry
                  Util.comment Util.comment_noindent
                  ldelim_re rdelim_re
                  ldelim_default rdelim_default

(* Variable: block_ldelim_re *)
let block_ldelim_re = /[ \t\n]+\{[ \t\n]*/

(* Variable: block_rdelim_re *)
let block_rdelim_re = /[ \t\n]*\}/

(* Variable: block_ldelim_default *)
let block_ldelim_default = " {\n"

(* Variable: block_rdelim_default *)
let block_rdelim_default = "}"

(************************************************************************
 * View: block
 *   A block enclosed in brackets
 *
 *   Parameters:
 *     entry:lens - the entry to be stored inside the block.
 *                  This entry should not include <Util.empty>,
 *                  <Util.comment> or <Util.comment_noindent>,
 *                  should not be indented or finish with an eol.
 ************************************************************************)
let block (entry:lens) = block_setdelim entry
                         block_ldelim_re block_rdelim_re
                         block_ldelim_default block_rdelim_default

(* Variable: block_ldelim_newlines_re *)
let block_ldelim_newlines_re = /[ \t\n]*\{([ \t\n]*\n)?/

(* Variable: block_rdelim_newlines_re *)
let block_rdelim_newlines_re = /[ \t]*\}/

(* Variable: block_ldelim_newlines_default *)
let block_ldelim_newlines_default = "\n{\n"

(* Variable: block_rdelim_newlines_default *)
let block_rdelim_newlines_default = "}"

(************************************************************************
 * View: block_newline
 *   A block enclosed in brackets, with newlines forced
 *   and indentation defaulting to a tab.
 *
 *   Parameters:
 *     entry:lens - the entry to be stored inside the block.
 *                  This entry should not include <Util.empty>,
 *                  <Util.comment> or <Util.comment_noindent>,
 *                  should be indented and finish with an eol.
 ************************************************************************)
let block_newlines (entry:lens) (comment:lens) =
   del block_ldelim_newlines_re block_ldelim_newlines_default
 . ((entry | comment) . (Util.empty | entry | comment)*)?
 . del block_rdelim_newlines_re block_rdelim_newlines_default

(************************************************************************
 * View: block_newlines_spc
 *   A block enclosed in brackets, with newlines forced
 *   and indentation defaulting to a tab. The opening brace
 *   must be preceded by whitespace
 *
 *   Parameters:
 *     entry:lens - the entry to be stored inside the block.
 *                  This entry should not include <Util.empty>,
 *                  <Util.comment> or <Util.comment_noindent>,
 *                  should be indented and finish with an eol.
 ************************************************************************)
let block_newlines_spc (entry:lens) (comment:lens) =
   del (/[ \t\n]/ . block_ldelim_newlines_re) block_ldelim_newlines_default
 . ((entry | comment) . (Util.empty | entry | comment)*)?
 . del block_rdelim_newlines_re block_rdelim_newlines_default

(************************************************************************
 * View: named_block
 *   A named <block> enclosed in brackets
 *
 *   Parameters:
 *     kw:regexp  - the regexp for the block name
 *     entry:lens - the entry to be stored inside the block
 *                   this entry should not include <Util.empty>
 ************************************************************************)
let named_block (kw:regexp) (entry:lens) = [ key kw . block entry . eol ]


(************************************************************************
 * Group:               COMBINATORICS
 ************************************************************************)

(************************************************************************
 * View: combine_two_ord
 *   Combine two lenses, ensuring first lens is first
 *
 *   Parameters:
 *     a:lens - the first lens
 *     b:lens - the second lens
 ************************************************************************)
let combine_two_ord (a:lens) (b:lens) = a . b

(************************************************************************
 * View: combine_two
 *   Combine two lenses
 *
 *   Parameters:
 *     a:lens - the first lens
 *     b:lens - the second lens
 ************************************************************************)
let combine_two (a:lens) (b:lens) =
  combine_two_ord a b | combine_two_ord b a

(************************************************************************
 * View: combine_two_opt_ord
 *   Combine two lenses optionally, ensuring first lens is first
 *   (a, and optionally b)
 *
 *   Parameters:
 *     a:lens - the first lens
 *     b:lens - the second lens
 ************************************************************************)
let combine_two_opt_ord (a:lens) (b:lens) = a . b?

(************************************************************************
 * View: combine_two_opt
 *   Combine two lenses optionally
 *   (either a, b, or both, in any order)
 *
 *   Parameters:
 *     a:lens - the first lens
 *     b:lens - the second lens
 ************************************************************************)
let combine_two_opt (a:lens) (b:lens) =
  combine_two_opt_ord a b | combine_two_opt_ord b a

(************************************************************************
 * View: combine_three_ord
 *   Combine three lenses, ensuring first lens is first
 *   (a followed by either b, c, in any order)
 *
 *   Parameters:
 *     a:lens - the first lens
 *     b:lens - the second lens
 *     c:lens - the third lens
 ************************************************************************)
let combine_three_ord (a:lens) (b:lens) (c:lens) =
  combine_two_ord a (combine_two b c)

(************************************************************************
 * View: combine_three
 *   Combine three lenses
 *
 *   Parameters:
 *     a:lens - the first lens
 *     b:lens - the second lens
 *     c:lens - the third lens
 ************************************************************************)
let combine_three (a:lens) (b:lens) (c:lens) =
    combine_three_ord a b c
  | combine_three_ord b a c
  | combine_three_ord c b a


(************************************************************************
 * View: combine_three_opt_ord
 *   Combine three lenses optionally, ensuring first lens is first
 *   (a followed by either b, c, or any of them, in any order)
 *
 *   Parameters:
 *     a:lens - the first lens
 *     b:lens - the second lens
 *     c:lens - the third lens
 ************************************************************************)
let combine_three_opt_ord (a:lens) (b:lens) (c:lens) =
  combine_two_opt_ord a (combine_two_opt b c)

(************************************************************************
 * View: combine_three_opt
 *   Combine three lenses optionally
 *   (either a, b, c, or any of them, in any order)
 *
 *   Parameters:
 *     a:lens - the first lens
 *     b:lens - the second lens
 *     c:lens - the third lens
 ************************************************************************)
let combine_three_opt (a:lens) (b:lens) (c:lens) =
    combine_three_opt_ord a b c
  | combine_three_opt_ord b a c
  | combine_three_opt_ord c b a
