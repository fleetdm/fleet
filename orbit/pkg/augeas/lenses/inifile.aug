(*
Module: IniFile
  Generic module to create INI files lenses

Author: Raphael Pinson <raphink@gmail.com>

About: License
  This file is licensed under the LGPL v2+, like the rest of Augeas.

About: TODO
  Things to add in the future
  - Support double quotes in value

About: Lens usage
  This lens is made to provide generic primitives to construct INI File lenses.
  See <Puppet>, <PHP>, <MySQL> or <Dput> for examples of real life lenses using it.

About: Examples
  The <Test_IniFile> file contains various examples and tests.
*)

module IniFile  =


(************************************************************************
 * Group:               USEFUL PRIMITIVES
 *************************************************************************)

(* Group: Internal primitives *)

(*
Variable: eol
  End of line, inherited from <Util.eol>
*)
let eol = Util.doseol


(* Group: Separators *)



(*
Variable: sep
  Generic separator

  Parameters:
    pat:regexp - the pattern to delete
    default:string - the default string to use
*)
let sep (pat:regexp) (default:string)
                       = Sep.opt_space . del pat default

(*
Variable: sep_noindent
  Generic separator, no indentation

  Parameters:
    pat:regexp - the pattern to delete
    default:string - the default string to use
*)
let sep_noindent (pat:regexp) (default:string)
                       = del pat default

(*
Variable: sep_re
  The default regexp for a separator
*)

let sep_re             = /[=:]/

(*
Variable: sep_default
  The default separator value
*)
let sep_default        = "="


(* Group: Stores *)


(*
Variable: sto_to_eol
  Store until end of line
*)
let sto_to_eol         = Sep.opt_space . store Rx.space_in

(*
Variable: to_comment_re
  Regex until comment
*)
let to_comment_re = /[^";# \t\n][^";#\n]*[^";# \t\n]|[^";# \t\n]/

(*
Variable: sto_to_comment
  Store until comment
*)
let sto_to_comment = Sep.opt_space . store to_comment_re

(*
Variable: sto_multiline
  Store multiline values
*)
let sto_multiline = Sep.opt_space
         . store (to_comment_re
               . (/[ \t]*\n/ . Rx.space . to_comment_re)*)

(*
Variable: sto_multiline_nocomment
  Store multiline values without an end-of-line comment
*)
let sto_multiline_nocomment = Sep.opt_space
         . store (Rx.space_in . (/[ \t]*\n/ . Rx.space . Rx.space_in)*)


(* Group: Define comment and defaults *)

(*
View: comment_noindent
  Map comments into "#comment" nodes,
  no indentation allowed

  Parameters:
    pat:regexp - pattern to delete before commented data
    default:string - default pattern before commented data

  Sample Usage:
  (start code)
    let comment  = IniFile.comment_noindent "#" "#"
    let comment  = IniFile.comment_noindent IniFile.comment_re IniFile.comment_default
  (end code)
*)
let comment_noindent (pat:regexp) (default:string) =
  Util.comment_generic_seteol (pat . Rx.opt_space) default eol

(*
View: comment
  Map comments into "#comment" nodes

  Parameters:
    pat:regexp - pattern to delete before commented data
    default:string - default pattern before commented data

  Sample Usage:
  (start code)
    let comment  = IniFile.comment "#" "#"
    let comment  = IniFile.comment IniFile.comment_re IniFile.comment_default
  (end code)
*)
let comment (pat:regexp) (default:string) =
  Util.comment_generic_seteol (Rx.opt_space . pat . Rx.opt_space) default eol

(*
Variable: comment_re
  Default regexp for <comment> pattern
*)

let comment_re         = /[;#]/

(*
Variable: comment_default
  Default value for <comment> pattern
*)
let comment_default    = ";"

(*
View: empty_generic
  Empty line, including empty comments

  Parameters:
    indent:regexp     - the indentation regexp
    comment_re:regexp - the comment separator regexp
*)
let empty_generic (indent:regexp) (comment_re:regexp) =
  Util.empty_generic_dos (indent . comment_re? . Rx.opt_space)

(*
View: empty
  Empty line
*)
let empty = empty_generic Rx.opt_space comment_re

(*
View: empty_noindent
  Empty line, without indentation
*)
let empty_noindent = empty_generic "" comment_re


(************************************************************************
 * Group:                     ENTRY
 *************************************************************************)

(* Group: entry includes comments *)

(*
View: entry_generic_nocomment
  A very generic INI File entry, not including comments
  It allows to set the key lens (to set indentation
  or subnodes linked to the key) as well as the comment
  separator regexp, used to tune the store regexps.

  Parameters:
    kw:lens           - lens to match the key, including optional indentation
    sep:lens          - lens to use as key/value separator
    comment_re:regexp - comment separator regexp
    comment:lens      - lens to use as comment

  Sample Usage:
     > let entry = IniFile.entry_generic (key "setting") sep IniFile.comment_re comment
*)
let entry_generic_nocomment (kw:lens) (sep:lens)
                            (comment_re:regexp) (comment:lens) =
     let bare_re_noquot = (/[^" \t\r\n]/ - comment_re)
  in let bare_re = (/[^\r\n]/ - comment_re)+
  in let no_quot = /[^"\r\n]*/
  in let bare = Quote.do_dquote_opt_nil (store (bare_re_noquot . (bare_re* . bare_re_noquot)?))
  in let quoted = Quote.do_dquote (store (no_quot . comment_re+ . no_quot))
  in [ kw . sep . (Sep.opt_space . bare)? . (comment|eol) ]
   | [ kw . sep . Sep.opt_space . quoted . (comment|eol) ]

(*
View: entry_generic
  A very generic INI File entry
  It allows to set the key lens (to set indentation
  or subnodes linked to the key) as well as the comment
  separator regexp, used to tune the store regexps.

  Parameters:
    kw:lens           - lens to match the key, including optional indentation
    sep:lens          - lens to use as key/value separator
    comment_re:regexp - comment separator regexp
    comment:lens      - lens to use as comment

  Sample Usage:
     > let entry = IniFile.entry_generic (key "setting") sep IniFile.comment_re comment
*)
let entry_generic (kw:lens) (sep:lens) (comment_re:regexp) (comment:lens) =
  entry_generic_nocomment kw sep comment_re comment | comment

(*
View: entry
  Generic INI File entry

  Parameters:
    kw:regexp    - keyword regexp for the label
    sep:lens     - lens to use as key/value separator
    comment:lens - lens to use as comment

  Sample Usage:
     > let entry = IniFile.entry setting sep comment
*)
let entry (kw:regexp) (sep:lens) (comment:lens) =
     entry_generic (key kw) sep comment_re comment

(*
View: indented_entry
  Generic INI File entry that might be indented with an arbitrary
  amount of whitespace

  Parameters:
    kw:regexp    - keyword regexp for the label
    sep:lens     - lens to use as key/value separator
    comment:lens - lens to use as comment

  Sample Usage:
     > let entry = IniFile.indented_entry setting sep comment
*)
let indented_entry (kw:regexp) (sep:lens) (comment:lens) =
     entry_generic (Util.indent . key kw) sep comment_re comment

(*
View: entry_multiline_generic
  A very generic multiline INI File entry
  It allows to set the key lens (to set indentation
  or subnodes linked to the key) as well as the comment
  separator regexp, used to tune the store regexps.

  Parameters:
    kw:lens           - lens to match the key, including optional indentation
    sep:lens          - lens to use as key/value separator
    comment_re:regexp - comment separator regexp
    comment:lens      - lens to use as comment
    eol:lens          - lens for end of line

  Sample Usage:
     > let entry = IniFile.entry_generic (key "setting") sep IniFile.comment_re comment comment_or_eol
*)
let entry_multiline_generic (kw:lens) (sep:lens) (comment_re:regexp)
                            (comment:lens) (eol:lens) =
     let newline = /\r?\n[ \t]+/
  in let bare =
          let word_re_noquot = (/[^" \t\r\n]/ - comment_re)+
       in let word_re = (/[^\r\n]/ - comment_re)+
       in let base_re = (word_re_noquot . (word_re* . word_re_noquot)?)
       in let sto_re = base_re . (newline . base_re)*
                     | (newline . base_re)+
       in Quote.do_dquote_opt_nil (store sto_re)
  in let quoted =
          let no_quot = /[^"\r\n]*/
       in let base_re = (no_quot . comment_re+ . no_quot)
       in let sto_re = base_re . (newline . base_re)*
                     | (newline . base_re)+
       in Quote.do_dquote (store sto_re)
  in [ kw . sep . (Sep.opt_space . bare)? . eol ]
   | [ kw . sep . Sep.opt_space . quoted . eol ]
   | comment
  

(*
View: entry_multiline
  Generic multiline INI File entry

  Parameters:
    kw:regexp    - keyword regexp for the label
    sep:lens     - lens to use as key/value separator
    comment:lens - lens to use as comment
*)
let entry_multiline (kw:regexp) (sep:lens) (comment:lens) =
  entry_multiline_generic (key kw) sep comment_re comment (comment|eol)

(*
View: entry_multiline_nocomment
  Generic multiline INI File entry without an end-of-line comment

  Parameters:
    kw:regexp    - keyword regexp for the label
    sep:lens     - lens to use as key/value separator
    comment:lens - lens to use as comment
*)
let entry_multiline_nocomment (kw:regexp) (sep:lens) (comment:lens) =
  entry_multiline_generic (key kw) sep comment_re comment eol

(*
View: entry_list
  Generic INI File list entry

  Parameters:
    kw:regexp     - keyword regexp for the label
    sep:lens      - lens to use as key/value separator
    sto:regexp    - store regexp for the values
    list_sep:lens - lens to use as list separator
    comment:lens  - lens to use as comment
*)
let entry_list (kw:regexp) (sep:lens) (sto:regexp) (list_sep:lens) (comment:lens) =
  let list = counter "elem"
      . Build.opt_list [ seq "elem" . store sto ] list_sep
  in Build.key_value_line_comment kw sep (Sep.opt_space . list) comment

(*
View: entry_list_nocomment
  Generic INI File list entry without an end-of-line comment

  Parameters:
    kw:regexp     - keyword regexp for the label
    sep:lens      - lens to use as key/value separator
    sto:regexp    - store regexp for the values
    list_sep:lens - lens to use as list separator
*)
let entry_list_nocomment (kw:regexp) (sep:lens) (sto:regexp) (list_sep:lens) =
  let list = counter "elem"
      . Build.opt_list [ seq "elem" . store sto ] list_sep
  in Build.key_value_line kw sep (Sep.opt_space . list)

(*
Variable: entry_re
  Default regexp for <entry> keyword
*)
let entry_re           = ( /[A-Za-z][A-Za-z0-9._-]*/ )


(************************************************************************
 * Group:                      RECORD
 *************************************************************************)

(* Group: Title definition *)

(*
View: title
  Title for <record>. This maps the title of a record as a node in the abstract tree.

  Parameters:
    kw:regexp - keyword regexp for the label

  Sample Usage:
    > let title   = IniFile.title IniFile.record_re
*)
let title (kw:regexp)
                       = Util.del_str "[" . key kw
                         . Util.del_str "]". eol

(*
View: indented_title
  Title for <record>. This maps the title of a record as a node in the abstract tree. The title may be indented with arbitrary amounts of whitespace

  Parameters:
    kw:regexp - keyword regexp for the label

  Sample Usage:
    > let title   = IniFile.title IniFile.record_re
*)
let indented_title (kw:regexp)
                       = Util.indent . title kw

(*
View: title_label
  Title for <record>. This maps the title of a record as a value in the abstract tree.

  Parameters:
    name:string - name for the title label
    kw:regexp   - keyword regexp for the label

  Sample Usage:
    > let title   = IniFile.title_label "target" IniFile.record_label_re
*)
let title_label (name:string) (kw:regexp)
                       = label name
                         . Util.del_str "[" . store kw
                         . Util.del_str "]". eol

(*
View: indented_title_label
  Title for <record>. This maps the title of a record as a value in the abstract tree. The title may be indented with arbitrary amounts of whitespace

  Parameters:
    name:string - name for the title label
    kw:regexp   - keyword regexp for the label

  Sample Usage:
    > let title   = IniFile.title_label "target" IniFile.record_label_re
*)
let indented_title_label (name:string) (kw:regexp)
                       = Util.indent . title_label name kw


(*
Variable: record_re
  Default regexp for <title> keyword pattern
*)
let record_re          = ( /[^]\r\n\/]+/ - /#comment/ )

(*
Variable: record_label_re
  Default regexp for <title_label> keyword pattern
*)
let record_label_re    = /[^]\r\n]+/


(* Group: Record definition *)

(*
View: record_noempty
  INI File Record with no empty lines allowed.

  Parameters:
    title:lens - lens to use for title. Use either <title> or <title_label>.
    entry:lens - lens to use for entries in the record. See <entry>.
*)
let record_noempty (title:lens) (entry:lens)
                       = [ title
		       . entry* ]

(*
View: record
  Generic INI File record

  Parameters:
    title:lens - lens to use for title. Use either <title> or <title_label>.
    entry:lens - lens to use for entries in the record. See <entry>.

  Sample Usage:
    > let record  = IniFile.record title entry
*)
let record (title:lens) (entry:lens)
                       = record_noempty title ( entry | empty )


(************************************************************************
 * Group:                      GENERIC LENSES
 *************************************************************************)


(*

Group: Lens definition

View: lns_noempty
  Generic INI File lens with no empty lines

  Parameters:
    record:lens  - record lens to use. See <record_noempty>.
    comment:lens - comment lens to use. See <comment>.

  Sample Usage:
    > let lns     = IniFile.lns_noempty record comment
*)
let lns_noempty (record:lens) (comment:lens)
                       = comment* . record*

(*
View: lns
  Generic INI File lens

  Parameters:
    record:lens  - record lens to use. See <record>.
    comment:lens - comment lens to use. See <comment>.

  Sample Usage:
    > let lns     = IniFile.lns record comment
*)
let lns (record:lens) (comment:lens)
                       = lns_noempty record (comment|empty)


(************************************************************************
 * Group:                   READY-TO-USE LENSES
 *************************************************************************)

let record_anon (entry:lens) = [ label "section" . value ".anon" . ( entry | empty )+ ]

(*
View: lns_loose
  A loose, ready-to-use lens, featuring:
    - sections as values (to allow '/' in names)
    - support empty lines and comments
    - support for [#;] as comment, defaulting to ";"
    - .anon sections
    - don't allow multiline values
    - allow indented titles
    - allow indented entries
*)
let lns_loose = 
     let l_comment = comment comment_re comment_default
  in let l_sep = sep sep_re sep_default
  in let l_entry = indented_entry entry_re l_sep l_comment
  in let l_title = indented_title_label "section" (record_label_re - ".anon")
  in let l_record = record l_title l_entry
  in (record_anon l_entry)? . l_record*

(*
View: lns_loose_multiline
  A loose, ready-to-use lens, featuring:
    - sections as values (to allow '/' in names)
    - support empty lines and comments
    - support for [#;] as comment, defaulting to ";"
    - .anon sections
    - allow multiline values
*)
let lns_loose_multiline = 
     let l_comment = comment comment_re comment_default
  in let l_sep = sep sep_re sep_default
  in let l_entry = entry_multiline entry_re l_sep l_comment
  in let l_title = title_label "section" (record_label_re - ".anon")
  in let l_record = record l_title l_entry
  in (record_anon l_entry)? . l_record*

