(*
Module: Chrony
  Parses the chrony config file

Author: Pat Riehecky <riehecky@fnal.gov>

About: Reference
  This lens tries to keep as close as possible to chrony config syntax

  See http://chrony.tuxfamily.org/manual.html#Configuration-file

About: License
  This file is licenced under the LGPL v2+, like the rest of Augeas.

About: Lens Usage
  To be documented

About: Configuration files
  This lens applies to /etc/chrony.conf

  See <filter>.
*)

module Chrony =
  autoload xfm

(************************************************************************
 * Group: Import provided expressions
 ************************************************************************)
    (* View: empty *)
    let empty   = Util.empty

    (* View: eol *)
    let eol     = Util.eol

    (* View: space *)
    let space   = Sep.space

    (* Variable: email_addr *)
    let email_addr = Rx.email_addr

    (* Variable: word *)
    let word       = Rx.word

    (* Variable: integer *)
    let integer    = Rx.relinteger

    (* Variable: decimal *)
    let decimal    = Rx.reldecimal

    (* Variable: ip *)
    let ip         = Rx.ip

    (* Variable: path *)
    let path       = Rx.fspath

(************************************************************************
 * Group: Create required expressions
 ************************************************************************)
    (* Variable: hex *)
    let hex = /[0-9a-fA-F]+/

    (* Variable: number *)
    let number = integer | decimal | decimal . /[eE]/ . integer | hex

    (* Variable: address_re *)
    let address_re = Rx.ip | Rx.hostname

    (*
       View: comment
            from 4.2.1 of the upstream doc
            Chrony comments start with: ! ; # or % and must be on their own line
    *)
    let comment = Util.comment_generic /[ \t]*[!;#%][ \t]*/ "# "

    (* Variable: no_space
         No spaces or comment characters
    *)
    let no_space   = /[^ \t\r\n!;#%]+/

    (* Variable: cmd_options
         Server/Peer/Pool options with values
    *)
    let cmd_options = "asymmetry"
                    | "certset"
                    | "extfield"
                    | "filter"
                    | "key"
                    | /maxdelay((dev)?ratio)?/
                    | /(min|max)poll/
                    | /(min|max)samples/
                    | "maxsources"
                    | "mindelay"
                    | "offset"
                    | "polltarget"
                    | "port"
                    | "presend"
                    | "version"

    (* Variable: cmd_flags
         Server/Peer/Pool options without values
    *)
    let cmd_flags = "auto_offline"|"iburst"|"noselect"|"offline"|"prefer"
                  |"copy"|"require"|"trust"|"xleave"|"burst"|"nts"

    (* Variable: ntp_source
         Server/Peer/Pool key names
    *)
    let ntp_source = "server"|"peer"|"pool"

    (* Variable: allowdeny_types
         Key names for access configuration
    *)
    let allowdeny_types = "allow"|"deny"|"cmdallow"|"cmddeny"

    (* Variable: hwtimestamp_options
         HW timestamping options with values
    *)
    let hwtimestamp_options = "minpoll"|"precision"|"rxcomp"|"txcomp"
                            |"minsamples"|"maxsamples"|"rxfilter"

    (* Variable: hwtimestamp_flags
         HW timestamping options without values
    *)
    let hwtimestamp_flags = "nocrossts"

    (* Variable: local_options
         local options with values
    *)
    let local_options = "stratum"|"distance"

    (* Variable: local_flags
         local options without values
    *)
    let local_flags = "orphan"

    (* Variable: ratelimit_options
         Rate limiting options with values
    *)
    let ratelimit_options = "interval"|"burst"|"leak"

    (* Variable: refclock_options
         refclock options with values
    *)
    let refclock_options = "refid"|"lock"|"poll"|"dpoll"|"filter"|"rate"
                            |"minsamples"|"maxsamples"|"offset"|"delay"
                            |"precision"|"maxdispersion"|"stratum"|"width"

    (* Variable: refclock_flags
         refclock options without values
    *)
    let refclock_flags = "noselect"|"pps"|"prefer"|"require"|"tai"|"trust"

    (* Variable: flags
         Options without values
    *)
    let flags = "dumponexit"
              | "generatecommandkey"
              | "lock_all"
              | "manual"
              | "noclientlog"
              | "nosystemcert"
              | "rtconutc"
              | "rtcsync"

    (* Variable: log_flags
        log has a specific options list
    *)
    let log_flags = "measurements"|"rawmeasurements"|"refclocks"|"rtc"
                  |"statistics"|"tempcomp"|"tracking"

    (* Variable: simple_keys
         Options with single values
    *)
    let simple_keys = "acquisitionport" | "authselectmode" | "bindacqaddress"
                    | "bindaddress" | "bindcmdaddress" | "bindacqdevice"
                    | "bindcmddevice" | "binddevice" | "clientloglimit"
                    | "clockprecision" | "combinelimit" | "commandkey"
                    | "cmdport" | "corrtimeratio" | "driftfile"
                    | "dscp"
                    | "dumpdir" | "hwclockfile" | "include" | "keyfile"
                    | "leapsecmode" | "leapsectz" | "linux_freq_scale"
                    | "linux_hz" | "logbanner" | "logchange" | "logdir"
                    | "maxclockerror" | "maxdistance" | "maxdrift"
                    | "maxjitter" | "maxsamples" | "maxslewrate"
                    | "maxntsconnections"
                    | "maxupdateskew" | "minsamples" | "minsources"
                    | "nocerttimecheck" | "ntsdumpdir" | "ntsntpserver"
                    | "ntsport" | "ntsprocesses" | "ntsrefresh" | "ntsrotate"
                    | "ntsservercert" | "ntsserverkey" | "ntstrustedcerts"
                    | "ntpsigndsocket" | "pidfile" | "ptpport"
                    | "port" | "reselectdist" | "rtcautotrim" | "rtcdevice"
                    | "rtcfile" | "sched_priority" | "stratumweight" | "user"

(************************************************************************
 * Group: Make some sub-lenses for use in later lenses
 ************************************************************************)
    (* View: host_flags *)
    let host_flags = [ space . key cmd_flags ]
    (* View: host_options *)
    let host_options = [ space . key cmd_options . space . store number ]
    (* View: log_flag_list *)
    let log_flag_list = [ space . key log_flags ]
    (* View: store_address *)
    let store_address = [ label "address" . store address_re ]

(************************************************************************
 * Group: Lenses for parsing out sections
 ************************************************************************)
    (* View: all_flags
        options without any arguments
    *)
    let all_flags = [ Util.indent . key flags . eol ]

    (* View: kv
        options with only one arg can be directly mapped to key = value
    *)
    let kv = [ Util.indent . key simple_keys . space . (store no_space) . eol ]

    (* Property: Options with multiple values
    
      Each of these gets their own parsing block
      - server|peer|pool <address> <options>
      - allow|deny|cmdallow|cmddeny [all] [<address[/subnet]>]
      - log <options>
      - broadcast <interval> <address> <optional port>
      - fallbackdrift <min> <max>
      - hwtimestamp <interface> <options>
      - initstepslew <threshold> <addr> <optional extra addrs>
      - local <options>
      - mailonchange <emailaddress> <threshold>
      - makestep <threshold> <limit>
      - maxchange <threshold> <delay> <limit>
      - ratelimit|cmdratelimit|ntsratelimit <options>
      - refclock <driver> <parameter> <options>
      - smoothtime <maxfreq> <maxwander> <options>
      - tempcomp <sensorfile> <interval> (<t0> <k0> <k1> <k2> | <pointfile> )
      - confdir|sourcedir <directories>
    *)

    (* View: host_list
        Find all NTP sources and their flags/options
    *)
    let host_list = [ Util.indent . key ntp_source
                         . space . store address_re
                         . ( host_flags | host_options )*
                         . eol ]

    (* View: allowdeny
        allow/deny/cmdallow/cmddeny has a specific syntax
    *)
    let allowdeny = [ Util.indent . key allowdeny_types
                        . [ space . key "all" ]?
                        . ( space . store ( no_space - "all" ) )?
                        . eol ]

    (* View: log_list
        log has a specific options list
    *)
    let log_list = [ Util.indent . key "log" . log_flag_list+ . eol ]

    (* View: bcast
         broadcast has specific syntax
    *)
    let bcast = [ Util.indent . key "broadcast"
                      . space . [ label "interval" . store integer ]
                      . space . store_address
                      . ( space . [ label "port" . store integer ] )?
                      . eol ]

    (* View: bcast
         confdir and sourcedir have specific syntax
    *)
    let dir_list = [ Util.indent . key /(conf|source)dir/
                      . [ label "directory" . space . store no_space ]+
                      . eol ]

    (* View: fdrift
         fallbackdrift has specific syntax
    *)
    let fdrift = [ Util.indent . key "fallbackdrift"
                      . space . [ label "min" . store integer ]
                      . space . [ label "max" . store integer ]
                      . eol ]

    (* View: hwtimestamp
         hwtimestamp has specific syntax
    *)
    let hwtimestamp = [ Util.indent . key "hwtimestamp"
                      . space . [ label "interface" . store no_space ]
                      . ( space . ( [ key hwtimestamp_flags ]
                         | [ key hwtimestamp_options . space
                             . store no_space ] )
                        )*
                      . eol ]
    (* View: istepslew
         initstepslew has specific syntax
    *)
    let istepslew = [ Util.indent . key "initstepslew" 
                         . space . [ label "threshold" . store number ]
                         . ( space . store_address )+
                         . eol ]

    (* View: local
         local has specific syntax
    *)
    let local = [ Util.indent . key "local"
                      . ( space . ( [ key local_flags ]
                         | [ key local_options . space . store no_space ] )
                        )*
                      . eol ]

    (* View: email
         mailonchange has specific syntax
    *)
    let email = [ Util.indent . key "mailonchange" . space
                     . [ label "emailaddress" . store email_addr ]
                     . space
                     . [ label "threshold" . store number ]
                     . eol ]

    (* View: makestep
         makestep has specific syntax
    *)
    let makestep = [ Util.indent . key "makestep"
                      . space
                      . [ label "threshold" . store number ]
                      . space
                      . [ label "limit" . store integer ]
                      . eol ]

    (* View: maxchange
         maxchange has specific syntax
    *)
    let maxchange = [ Util.indent . key "maxchange"
                      . space
                      . [ label "threshold" . store number ]
                      . space
                      . [ label "delay" . store integer ]
                      . space
                      . [ label "limit" . store integer ]
                      . eol ]

    (* View: ratelimit
         ratelimit/cmdratelimit has specific syntax
    *)
    let ratelimit = [ Util.indent . key /(cmd|nts)?ratelimit/
                      . [ space . key ratelimit_options
                              . space . store no_space ]*
                      . eol ]
    (* View: refclock
         refclock has specific syntax
    *)
    let refclock = [ Util.indent . key "refclock"
                      . space
                      . [ label "driver" . store word ]
                      . space
                      . [ label "parameter" . store no_space ]
                      . ( space . ( [ key refclock_flags ]
                         | [ key refclock_options . space . store no_space ] )
                        )*
                      . eol ]

    (* View: smoothtime
         smoothtime has specific syntax
    *)
    let smoothtime = [ Util.indent . key "smoothtime"
                      . space
                      . [ label "maxfreq" . store number ]
                      . space
                      . [ label "maxwander" . store number ]
                      . ( space . [ key "leaponly" ] )?
                      . eol ]

    (* View: tempcomp
         tempcomp has specific syntax
    *)
    let tempcomp = [ Util.indent . key "tempcomp"
                      . space
                      . [ label "sensorfile" . store path ]
                      . space
                      . [ label "interval" . store number ]
                      . space
                      . ( [ label "t0" . store number ] . space
                              . [ label "k0" . store number ] . space
                              . [ label "k1" . store number ] . space
                              . [ label "k2" . store number ]
                              | [ label "pointfile" . store path ] )
                      . eol ]

(************************************************************************
 * Group: Final lense summary
 ************************************************************************)
(* View: settings
 *   All supported chrony settings
 *)
let settings = host_list | allowdeny | log_list | bcast | fdrift | istepslew
             | local | email | makestep | maxchange | refclock | smoothtime
             | dir_list | hwtimestamp | ratelimit | tempcomp | kv | all_flags

(*
 * View: lns
 *   The crony lens
 *)
let lns = ( empty | comment | settings )*

(* View: filter
 *   The files parsed by default
 *)
let filter = incl "/etc/chrony.conf"

let xfm = transform lns filter

