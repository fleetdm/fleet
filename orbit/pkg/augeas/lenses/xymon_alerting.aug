(*
Module: Xymon_Alerting
  Parses xymon alerting files 

Author: Francois Maillard <fmaillard@gmail.com>

About: Reference
  This lens tries to keep as close as possible to `man 5 alerts.cfg` where possible.

About: License
   This file is licenced under the LGPL v2+, like the rest of Augeas.

About: Lens Usage
   To be documented

About: Not supported
   File inclusion are not followed

About: Configuration files
   This lens applies to /etc/xymon/alerts.d/*.cfg and /etc/xymon/alerts.cfg. See <filter>.

About: Examples
   The <Test_Xymon_Alerting> file contains various examples and tests.
*)

module Xymon_Alerting =
    autoload xfm

    (************************************************************************
     * Group:                 USEFUL PRIMITIVES
     *************************************************************************)

    (* View: store_word *)
    let store_word  = store /[^ =\t\n#]+/

    (* View: comparison The greater and lesser than operators *)
    let comparison  = store /[<>]/

    (* View: equal *)
    let equal       = Sep.equal

    (* View: ws *)
    let ws          = Sep.space

    (* View: eol *)
    let eol         = Util.eol

    (* View: ws_or_eol *)
    let ws_or_eol   = del /([ \t]+|[ \t]*\n[ \t]*)/ " "

    (* View: comment *)
    let comment = Util.comment

    (* View: empty *)
    let empty   = Util.empty

    (* View: include *)
    let include         = [ key "include" . ws . store_word . eol ]

    (************************************************************************
     * Group:                 MACRO DEFINITION
     *************************************************************************)

    (* View: macrodefinition
         A string that starts with $ and that is assigned something *)
    let macrodefinition = [ key /\$[^ =\t\n#\/]+/ . Sep.space_equal . store Rx.space_in . eol ]


    (* View: flag
         A flag value *)
    let flag (kw:string) = Build.flag kw

    (* View: kw_word
         A key=value value *)
    let kw_word (kw:regexp) = Build.key_value kw equal store_word

    (************************************************************************
     * Group:                 FILTERS 
     *************************************************************************)

    (* View: page
         The (ex)?page filter definition *)
    let page      = kw_word /(EX)?PAGE/

    (* View: group
         The (ex)?group filter definition *)
    let group     = kw_word /(EX)?GROUP/

    (* View: host
         The (ex)?host filter definition *)
    let host      = kw_word /(EX)?HOST/

    (* View: service
         The (ex)?service filter definition *)
    let service   = kw_word /(EX)?SERVICE/

    (* View: color
         The color filter definition *)
    let color     = kw_word "COLOR"

    (* View: time
         The time filter definition *)
    let time      = kw_word "TIME"

    (* View: duration
         The duration filter definition *)
    let duration  = [ key "DURATION" . [ label "operator" . comparison ] . [ label "value" . store_word ] ]
    (* View: recover
         The recover filter definition *)
    let recover   = flag "RECOVER"
    (* View: notice
         The notice filter definition *)
    let notice    = flag "NOTICE"

    (* View: rule_filter
         Filters are made out of any of the above filter definitions *)
    let rule_filter = page | group | host | service
                    | color | time | duration | recover | notice

    (* View: filters
         One or more filters *)
    let filters = [ label "filters" . Build.opt_list rule_filter ws ]

    (* View: filters_opt
         Zero, one or more filters *)
    let filters_opt = [ label "filters" . (ws . Build.opt_list rule_filter ws)? ]

    (* View: kw_word_filters_opt
         A <kw_word> entry with optional filters *)
    let kw_word_filters_opt (kw:string) = [ key kw . equal . store_word . filters_opt ]

    (* View: flag_filters_opt
         A <flag> with optional filters *) 
    let flag_filters_opt (kw:string) = [ key kw . filters_opt ]

    (************************************************************************
     * Group:                 RECIPIENTS
     *************************************************************************)

    (* View: mail
         The mail recipient definition *)
    let mail      = [ key "MAIL" . ws . store_word . filters_opt ]

    (* View: script
         The script recipient definition *)
    let script    = [ key "SCRIPT" . ws . [ label "script" . store_word ]
                  . ws . [ label "recipient" . store_word ] . filters_opt ]

    (* View: ignore
         The ignore recipient definition *)
    let ignore    = flag_filters_opt "IGNORE"

    (* View: format
         The format recipient definition *)
    let format    = kw_word_filters_opt "FORMAT"

    (* View: repeat
         The repeat recipient definition *)
    let repeat    = kw_word_filters_opt "REPEAT"

    (* View: unmatched
         The unmatched recipient definition *)
    let unmatched = flag_filters_opt "UNMATCHED"

    (* View: stop
         The stop recipient definition *)
    let stop      = flag_filters_opt "STOP"

    (* View: macro
         The macro recipient definition *)
    let macro     = [ key /\$[^ =\t\n#\/]+/ . filters_opt ]

    (* View: recipient
         Recipients are made out of any of the above recipient definitions *)
    let recipient = mail | script | ignore | format | repeat | unmatched
                  | stop | macro

    let recipients = [ label "recipients" . Build.opt_list recipient ws_or_eol ]


    (************************************************************************
     * Group:                 RULES
     *************************************************************************)

    (* View: rule
         Rules are made of rule_filter and then recipients sperarated by a whitespace *)
    let rule = [ seq "rules" . filters . ws_or_eol . recipients . eol ] 

    (* View: lns
         The Xymon_Alerting lens *)
    let lns = ( rule | macrodefinition | include | empty | comment )*

    (* Variable: filter *)
    let filter = incl "/etc/xymon/alerts.d/*.cfg"
               . incl "/etc/xymon/alerts.cfg"
               . Util.stdexcl

    let xfm = transform lns filter

