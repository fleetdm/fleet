(*
Module: Pbuilder
 Parses /etc/pbuilderrc, /etc/pbuilder/pbuilderrc

Author: Raphael Pinson <raphink@gmail.com>

About: Reference
  Pbuilderrc is a standard shellvars file.

About: License
  This file is licensed under the LGPL v2+, like the rest of Augeas.

About: Configuration files
  This lens applies to /etc/pbuilderrc and /etc/pbuilder/pbuilderrc.
  See <filter>.
*)

module Pbuilder =

autoload xfm

(* View: filter
    The pbuilder conffiles *)
let filter = incl "/etc/pbuilder/pbuilderrc"
           . incl "/etc/pbuilderrc"

(* View: lns
    The pbuilder lens *)
let lns    = Shellvars.lns

let xfm    = transform lns filter
