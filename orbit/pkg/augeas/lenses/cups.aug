(*
Module: Cups
  Parses cups configuration files

Author: Raphael Pinson <raphink@gmail.com>

About: License
   This file is licenced under the LGPL v2+, like the rest of Augeas.

About: Lens Usage
   To be documented

About: Examples
   The <Test_Cups> file contains various examples and tests.
*)

module Cups =

autoload xfm

(* View: lns *)
let lns = Httpd.lns

(* Variable: filter *)
let filter = incl "/etc/cups/*.conf"

let xfm = transform lns filter
