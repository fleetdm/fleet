(*
Module: NagiosObjects
  Parses /etc/{nagios{3,},icinga}/objects/*.cfg

Authors: Sebastien Aperghis-Tramoni <sebastien@aperghis.net>
         Raphaël Pinson <raphink@gmail.com>

About: License
  This file is licenced under the LGPL v2+, like the rest of Augeas.

About: Lens Usage
  To be documented

About: Configuration files

  This lens applies to /etc/{nagios{3,},icinga}/objects/*.cfg. See <filter>.
*)

module NagiosObjects =
    autoload xfm

    (* basic atoms *)
    let eol = Util.eol
    let ws  = Sep.space

    let keyword      = key /[A-Za-z0-9_]+/

    (* optional, but preferred, whitespace *)
    let opt_ws = del Rx.opt_space " "

    (* define an empty line *)
    let empty = Util.empty

    (* define a comment *)
    let comment = Util.comment_generic /[ \t]*[#;][ \t]*/ "# "

    (* define a field *)
    let object_field    =
       let field_name      = keyword in
       let field_value     = store Rx.space_in in
          [ Util.indent . field_name . ws
                        . field_value . eol ]

    (* define an object *)
    let object_def  =
       let object_type = keyword in
          [ Util.indent
          . Util.del_str "define" . ws
          . object_type . opt_ws
          . Util.del_str "{" . eol
          . ( empty | comment | object_field )*
          . Util.indent . Util.del_str "}" . eol ]

    (* main structure *)
    let lns = ( empty | comment | object_def )*

    let filter = incl "/etc/nagios3/objects/*.cfg"
               . incl "/etc/nagios/objects/*.cfg"
	       . incl "/etc/icinga/objects/*.cfg"

    let xfm = transform lns filter

