(*
Module: NagiosConfig
  Parses /etc/{nagios{3,},icinga}/*.cfg

Authors: Sebastien Aperghis-Tramoni <sebastien@aperghis.net>
         Raphaël Pinson <raphink@gmail.com>

About: License
  This file is licenced under the LGPL v2+, like the rest of Augeas.

About: Lens Usage
  To be documented

About: Configuration files
  This lens applies to /etc/{nagios{3,},icinga}/*.cfg. See <filter>.
*)

module NagiosCfg =
autoload xfm

(************************************************************************
 * Group: Utility variables/functions
 ************************************************************************)
(* View: param_def
    define a field *)
let param_def =
     let space_in  = /[^ \t\n][^\n=]*[^ \t\n]|[^ \t\n]/
  in key /[A-Za-z0-9_]+/
   . Sep.opt_space . Sep.equal . Sep.opt_space
   . store space_in

(* View: macro_def
    Macro line, as used in resource.cfg *)
let macro_def =
     let macro = /\$[A-Za-z0-9]+\$/
       in let macro_decl = Rx.word | Rx.fspath
     in key macro . Sep.space_equal . store macro_decl

(************************************************************************
 * Group: Entries
 ************************************************************************)
(* View: param
    Params can have sub params *)
let param =
     [ Util.indent . param_def
     . [ Sep.space . param_def ]*
     . Util.eol ]

(* View: macro *)
let macro = [ Util.indent . macro_def . Util.eol ]

(************************************************************************
 * Group: Lens
 ************************************************************************)
(* View: entry
    Define the accepted entries, such as param for regular configuration
    files, and macro for resources.cfg .*)
let entry = param
	  | macro

(* View: lns
    main structure *)
let lns = ( Util.empty | Util.comment | entry )*

(* View: filter *)
let filter = incl "/etc/nagios3/*.cfg"
           . incl "/etc/nagios/*.cfg"
	   . incl "/etc/icinga/*.cfg"
	   . excl "/etc/nagios3/commands.cfg"
	   . excl "/etc/nagios/commands.cfg"
	   . excl "/etc/nagios/nrpe.cfg"
	   . incl "/etc/icinga/commands.cfg"

let xfm = transform lns filter
