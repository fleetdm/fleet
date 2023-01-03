(*
Module: Fonts
  Parses the /etc/fonts directory

Author: Raphael Pinson <raphink@gmail.com>

About: Reference
  This lens tries to keep as close as possible to `man 5 fonts-conf` where possible.

About: License
   This file is licenced under the LGPL v2+, like the rest of Augeas.

About: Lens Usage
   To be documented

About: Configuration files
   This lens applies to files in the /etc/fonts directory. See <filter>.

About: Examples
   The <Test_Fonts> file contains various examples and tests.
*)

module Fonts =

autoload xfm

(* View: lns *)
let lns = Xml.lns

(* Variable: filter *)
let filter = incl "/etc/fonts/fonts.conf"
           . incl "/etc/fonts/conf.avail/*"
           . incl "/etc/fonts/conf.d/*"
           . excl "/etc/fonts/*/README"
           . Util.stdexcl

let xfm = transform lns filter
