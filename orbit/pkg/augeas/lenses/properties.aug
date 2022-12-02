(* Augeas module for editing Java properties files
 Author: Craig Dunn <craig@craigdunn.org>

  Limitations:
   - doesn't support \ alone on a line
   - values are not unescaped
   - multi-line properties are broken down by line, and can't be replaced with a single line

  See format info: http://docs.oracle.com/javase/6/docs/api/java/util/Properties.html#load(java.io.Reader)
*)


module Properties =
  (* Define some basic primitives *)
  let empty            = Util.empty_generic_dos /[ \t]*[#!]?[ \t]*/
  let eol              = Util.doseol
  let hard_eol         = del /\r?\n/ "\n"
  let sepch            = del /([ \t]*(=|:)|[ \t])/ "="
  let sepspc           = del /[ \t]/ " "
  let sepch_ns         = del /[ \t]*(=|:)/ "="
  let sepch_opt        = del /[ \t]*(=|:)?[ \t]*/ "="
  let value_to_eol_ws  = store /(:|=)[^\r\n]*[^ \t\r\n\\]/
  let value_to_bs_ws   = store /(:|=)[^\n]*[^\\\n]/
  let value_to_eol     = store /([^ \t\n:=][^\n]*[^ \t\r\n\\]|[^ \t\r\n\\:=])/
  let value_to_bs      = store /([^ \t\n:=][^\n]*[^\\\n]|[^ \t\n\\:=])/
  let indent           = Util.indent
  let backslash        = del /[\\][ \t]*\n/ "\\\n"
  let opt_backslash    = del /([\\][ \t]*\n)?/ ""
  let entry            = /([^ \t\r\n:=!#\\]|[\\]:|[\\]=|[\\][\t ]|[\\][^\/\r\n])+/

  let multi_line_entry =
      [ indent . value_to_bs? . backslash ] .
      [ indent . value_to_bs . backslash ] * .
      [ indent . value_to_eol . eol ] . value " < multi > "

  let multi_line_entry_ws =
      opt_backslash .
      [ indent . value_to_bs_ws . backslash ] + .
      [ indent . value_to_eol . eol ] . value " < multi_ws > "

  (* define comments and properties*)
  let bang_comment     = [ label "!comment" . del /[ \t]*![ \t]*/ "! " . store /([^ \t\n].*[^ \t\r\n]|[^ \t\r\n])/ . eol ]
  let comment          = ( Util.comment | bang_comment )
  let property         = [ indent . key entry . sepch . ( multi_line_entry | indent . value_to_eol . eol ) ]
  let property_ws         = [ indent . key entry . sepch_ns . ( multi_line_entry_ws | indent . value_to_eol_ws . eol ) ]
  let empty_property   = [ indent . key entry . sepch_opt . hard_eol ]
  let empty_key        = [ sepch_ns . ( multi_line_entry | indent . value_to_eol . eol ) ]

  (* setup our lens and filter*)
  let lns              = ( empty | comment | property_ws | property | empty_property | empty_key ) *
