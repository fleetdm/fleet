(*
Module: Anacron
 Parses /etc/anacrontab

Author: Raphael Pinson <raphink@gmail.com>

About: Reference
 This lens tries to keep as close as possible to `man 5 anacrontab` where
 possible.

About: License
  This file is licensed under the LGPL v2+, like the rest of Augeas.

About: Lens Usage

About: Configuration files
  This lens applies to /etc/anacrontab. See <filter>.

About: Examples
   The <Test_Anacron> file contains various examples and tests.
*)

module Anacron =
  autoload xfm

(************************************************************************
 * Group:                       ENTRIES
 *************************************************************************)


(************************************************************************
 * View: shellvar
 *   A shell variable in crontab
 *************************************************************************)

let shellvar = Cron.shellvar


(* View: period *)
let period = [ label "period" . store Rx.integer ]

(* Variable: period_name_re
     The valid values for <period_name>. Currently only "monthly" *)
let period_name_re = "monthly"

(************************************************************************
 * View: period_name
 *   In the format "@keyword"
 *************************************************************************)
let period_name = [ label "period_name" . Util.del_str "@"
                  . store period_name_re ]

(************************************************************************
 * View: delay
 *   The delay for an <entry>
 *************************************************************************)
let delay = [ label "delay" . store Rx.integer ]

(************************************************************************
 * View: job_identifier
 *   The job_identifier for an <entry>
 *************************************************************************)
let job_identifier = [ label "job-identifier" . store Rx.word ]

(************************************************************************
 * View: entry
 *   An anacrontab entry
 *************************************************************************)

let entry = [ label "entry" . Util.indent
            . ( period | period_name )
            . Sep.space . delay
            . Sep.space . job_identifier
            . Sep.space . store Rx.space_in . Util.eol ]


(*
 * View: lns
 *   The anacron lens
 *)
let lns = ( Util.empty | Util.comment | shellvar | entry )*


(* Variable: filter *)
let filter = incl "/etc/anacrontab"

let xfm = transform lns filter
