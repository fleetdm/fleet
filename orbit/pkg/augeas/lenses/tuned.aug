(*
Module: Tuned
  Parses Tuned's configuration files

Author: Pat Riehecky <riehecky@fnal.gov>

About: Reference
    This lens is based on tuned's tuned-main.conf

About: License
  This file is licensed under the LGPL v2+, like the rest of Augeas.
*)

module Tuned =
autoload xfm

let lns = Simplevars.lns

let filter = incl "/etc/tuned/tuned-main.conf"

let xfm = transform lns filter
