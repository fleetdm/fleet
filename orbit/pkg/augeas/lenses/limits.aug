(* Limits module for Augeas
 Author: Free Ekanayaka <free@64studio.com>

 Reference: /etc/security/limits.conf

*)

module Limits =

   autoload xfm

(************************************************************************
 *                           USEFUL PRIMITIVES
 *************************************************************************)

let eol        = Util.eol
let comment_or_eol = Util.comment_or_eol
let spc        = Util.del_ws_spc
let comment    = Util.comment
let empty      = Util.empty

let sto_to_eol = store /([^ \t\n].*[^ \t\n]|[^ \t\n])/

(************************************************************************
 *                               ENTRIES
 *************************************************************************)

let domain     = label "domain" . store /[%@]?[A-Za-z0-9_.:-]+|\*/

let type_re    = "soft"
               | "hard"
               | "-"
let type       = [ label "type" . store type_re ]

let item_re    = "core"
               | "data"
               | "fsize"
               | "memlock"
               | "nofile"
               | "rss"
               | "stack"
               | "cpu"
               | "nproc"
               | "as"
               | "maxlogins"
               | "maxsyslogins"
               | "priority"
               | "locks"
               | "sigpending"
               | "msgqueue"
               | "nice"
               | "rtprio"
               | "chroot"
let item       = [ label "item" . store item_re ]

let value      = [ label "value" . store /[A-Za-z0-9_.\/-]+/ ]
let entry      = [ domain . spc
                 . type   . spc
                 . item   . spc
                 . value  . comment_or_eol ]

(************************************************************************
 *                                LENS
 *************************************************************************)

let lns        = (comment|empty|entry) *

let filter     = incl "/etc/security/limits.conf"
               . incl "/etc/security/limits.d/*.conf"

let xfm        = transform lns filter
