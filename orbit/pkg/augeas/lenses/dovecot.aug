(*
Module: Dovecot
  Parses dovecot configuration files.

Author: Serge Smetana <serge.smetana@gmail.com>
  Acunote http://www.acunote.com
  Pluron, Inc. http://pluron.com

About: License
  This file is licensed under the LGPL v2+.

About: Configuration files
  This lens applies to /etc/dovecot/dovecot.conf and files in
  /etc/dovecot/conf.d/. See <filter>.

About: Examples
  The <Test_Dovecot> file contains various examples and tests.

About: TODO
  Support for multiline values like queries in dict-sql.conf
*)

module Dovecot =

  autoload xfm

(******************************************************************
 * Group:                 USEFUL PRIMITIVES
 ******************************************************************)

(* View: indent *)
let indent = Util.indent

(* View: eol *)
let eol = Util.eol

(* View: empty
Map empty lines. *)
let empty = Util.empty

(* View: comment
Map comments in "#comment" nodes. *)
let comment = Util.comment

(* View: eq *)
let eq = del /[ \t]*=/ " ="

(* Variable: any *)
let any = Rx.no_spaces

(* Variable: value
Match any value after " =".
Should not start and end with spaces. May contain spaces inside *)
let value = any . (Rx.space . any)*

(* View: command_start *)
let command_start = Util.del_str "!"


(******************************************************************
 * Group:                        ENTRIES
 ******************************************************************)

(* Variable: commands *)
let commands = /include|include_try/

(* Variable: block_names *)
let block_names = /dict|userdb|passdb|protocol|service|plugin|namespace|map|fields|unix_listener|fifo_listener|inet_listener/

(* Variable: keys
Match any possible key except commands and block names. *)
let keys = Rx.word - (commands | block_names)

(* View: entry
Map simple "key = value" entries including "key =" entries with empty value. *)
let entry = [ indent . key keys. eq . (Sep.opt_space . store value)? . eol ]

(* View: command
Map commands started with "!". *)
let command = [ command_start . key commands . Sep.space . store Rx.fspath . eol ]

(*
View: dquote_spaces
  Make double quotes mandatory if value contains spaces,
  and optional if value doesn't contain spaces.

Based off Quote.dquote_spaces

Parameters:
  lns1:lens - the lens before
  lns2:lens - the lens after
*)
let dquote_spaces (lns1:lens) (lns2:lens) =
     (* bare has no spaces, and is optionally quoted *)
     let bare = Quote.do_dquote_opt (store /[^" \t\n]+/)
     (* quoted has at least one space, and must be quoted *)
  in let quoted = Quote.do_dquote (store /[^"\n]*[ \t]+[^"\n]*/)
  in [ lns1 . bare . lns2 ] | [ lns1 . quoted . lns2 ]

let mailbox = indent
            . dquote_spaces
               (key /mailbox/ . Sep.space)
               (Build.block_newlines_spc entry comment . eol)

let block_ldelim_newlines_re = /[ \t]+\{([ \t\n]*\n)?/

let block_newlines (entry:lens) (comment:lens) =
      let indent = del Rx.opt_space "\t"
   in del block_ldelim_newlines_re Build.block_ldelim_default
 . ((entry | comment) . (Util.empty | entry | comment)*)?
 . del Build.block_rdelim_newlines_re Build.block_rdelim_newlines_default

(* View: block
Map block enclosed in brackets recursively.
Block may be indented and have optional argument.
Block body may have entries, comments, empty lines, and nested blocks recursively. *)
let rec block = [ indent . key block_names . (Sep.space . Quote.do_dquote_opt (store /!?[\/A-Za-z0-9_-]+/))? . block_newlines (entry|block|mailbox) comment . eol ]


(******************************************************************
 * Group:                   LENS AND FILTER
 ******************************************************************)

(* View: lns
The Dovecot lens *)
let lns = (comment|empty|entry|command|block)*

(* Variable: filter *)
let filter = incl "/etc/dovecot/dovecot.conf"
           . (incl "/etc/dovecot/conf.d/*.conf")
           . incl "/usr/local/etc/dovecot/dovecot.conf"
           . (incl "/usr/local/etc/dovecot/conf.d/*.conf")
           . Util.stdexcl

let xfm = transform lns filter
