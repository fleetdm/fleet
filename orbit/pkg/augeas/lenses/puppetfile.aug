(*
Module: Puppetfile
  Parses libarian-puppet's Puppetfile format

Author: Raphael Pinson <raphael.pinson@camptocamp.com>

About: Reference
  See https://github.com/rodjek/librarian-puppet

About: License
   This file is licenced under the LGPL v2+, like the rest of Augeas.

About: Lens Usage
   To be documented

About: Configuration files
   This lens applies to Puppetfiles.

About: Examples
   The <Test_Puppetfile> file contains various examples and tests.
*)

module Puppetfile =

(* View: comma
     a comma, optionally preceded or followed by spaces or newlines *)
let comma = del /[ \t\n]*,[ \t\n]*/ ", "
let comma_nospace = del /[ \t\n]*,/ ","

let comment_or_eol = Util.eol | Util.comment_eol
let quote_to_comment_or_eol = Quote.do_quote (store /[^#\n]*/) . comment_or_eol

(* View: moduledir
     The moduledir setting specifies where modules from the Puppetfile will be installed *)
let moduledir = [ Util.indent . key "moduledir" . Sep.space
                . quote_to_comment_or_eol ]

(* View: forge
     a forge entry *)
let forge = [ Util.indent . key "forge" . Sep.space
            . quote_to_comment_or_eol ]

(* View: metadata
     a metadata entry *)
let metadata = [ Util.indent . key "metadata" . comment_or_eol ]

(* View: mod
     a module entry, with optional version and options *)
let mod =
     let mod_name = Quote.do_quote (store ((Rx.word . /[\/-]/)? . Rx.word))
  in let version = [ label "@version" . Quote.do_quote (store /[^#:\n]+/) . Util.comment_eol? ]
  in let sto_opt_val = store /[^#"', \t\n][^#"',\n]*[^#"', \t\n]|[^#"', \t\n]/
  in let opt = [
                 Util.del_str ":" . key Rx.word
                 . (del /[ \t]*=>[ \t]*/ " => " . Quote.do_quote_opt sto_opt_val)?
               ]
  in let opt_eol = del /([ \t\n]*\n)?/ ""
  in let opt_space_or_eol = del /[ \t\n]*/ " "
  in let comma_opt_eol_comment = comma_nospace . (opt_eol . Util.comment_eol)*
                               . opt_space_or_eol
  in let opts = Build.opt_list opt comma_opt_eol_comment
  in [ Util.indent . Util.del_str "mod" . seq "mod" . Sep.space . mod_name
     . (comma_opt_eol_comment . version)?
     . (comma_opt_eol_comment . opts . Util.comment_eol?)?
     . Util.eol ]

(* View: lns
     the Puppetfile lens *)
let lns = (Util.empty | Util.comment | forge | metadata | mod | moduledir )*
