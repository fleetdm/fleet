(*
Module: Cron_User
 Parses /var/spool/cron/*

Author: David Lutterkort <lutter@watzmann.net>

About: Reference
 This lens parses the user crontab files in /var/spool/cron. It produces
 almost the same tree as the Cron.lns, except that it never contains a user
 field.

About: License
  This file is licensed under the LGPL v2+, like the rest of Augeas.

About: Lens Usage
  Sample usage of this lens in augtool

    * Get the entry that launches '/usr/bin/ls'
      >  match '/files/var/spool/cron/foo/entry[. = "/usr/bin/ls"]'

About: Configuration files
  This lens applies to /var/spool/cron*. See <filter>.
 *)
module Cron_User =
  autoload xfm

(************************************************************************
 * View: entry
 *   A crontab entry for a user's crontab
 *************************************************************************)
let entry        = [ label "entry" . Cron.indent
                   . Cron.prefix?
                   . ( Cron.time | Cron.schedule )
                   . Cron.sep_spc . store Rx.space_in . Cron.eol ]

(*
 * View: lns
 *   The cron_user lens. Almost identical to Cron.lns
 *)
let lns = ( Cron.empty | Cron.comment | Cron.shellvar | entry )*

let filter =
  incl "/var/spool/cron/*" .
  Util.stdexcl

let xfm = transform lns filter
