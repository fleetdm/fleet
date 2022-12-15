(*
Module: Mailscanner_rules
  Parses MailScanner rules files.

Author: Andrew Colin Kissa <andrew@topdog.za.net>
  Baruwa Enterprise Edition http://www.baruwa.com

About: License
  This file is licensed under the LGPL v2+.

About: Configuration files
  This lens applies to MailScanner rules files
  The format is described below:

  # NOTE: Fields are separated by TAB characters --- Important!
  #
  # Syntax is allow/deny/deny+delete/rename/rename to replacement-text/email-addresses,
  #           then regular expression,
  #           then log text,
  #           then user report text.
  # The "email-addresses" can be a space or comma-separated list of email
  # addresses. If the rule hits, the message will be sent to these address(es)
  # instead of the original recipients.

  # If a rule is a "rename" rule, then the attachment filename will be renamed
  # according to the "Default Rename Pattern" setting in MailScanner.conf.
  # If a rule is a "rename" rule and the "to replacement-text" is supplied, then
  # the text matched by the regular expression in the 2nd field of the line
  # will be replaced with the "replacement-text" string.
  # For example, the rule
  # rename to .ppt	\.pps$	Renamed .pps to .ppt	Renamed .pps to .ppt
  # will find all filenames ending in ".pps" and rename them so they end in
  # ".ppt" instead.
*)

module Mailscanner_Rules =
autoload xfm

(************************************************************************
 *                           USEFUL PRIMITIVES
 *************************************************************************)

let eol = del /\n/ "\n"
let ws         = del /[\t]+/ "\t"
let comment    = Util.comment
let empty      = Util.empty
let action     = /allow|deny|deny\+delete|rename|rename[ ]+to[ ]+[^# \t\n]+|([A-Za-z0-9_+.-]+@[A-Za-z0-9_.-]+[, ]?)+/
let non_space  = /[^# \t\n]+/
let non_tab    = /[^\t\n]+/

let field (l:string) (r:regexp)
               = [ label l . store r ]

(************************************************************************
 *                               ENTRIES
 *************************************************************************)

let entry     = [ seq "rule" . field "action" action
                . ws . field "regex" non_tab
                . ws . field "log-text" non_tab
                . ws . field "user-report" non_tab
                . eol ]

(************************************************************************
 *                                LENS
 *************************************************************************)

let lns        = (comment|empty|entry)*

let filter     = (incl "/etc/MailScanner/filename.rules.conf")
                . (incl "/etc/MailScanner/filetype.rules.conf")
                . (incl "/etc/MailScanner/archives.filename.rules.conf")
                . (incl "/etc/MailScanner/archives.filetype.rules.conf")

let xfm        = transform lns filter
