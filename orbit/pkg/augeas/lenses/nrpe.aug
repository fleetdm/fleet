(*
Module: Nrpe
  Parses nagios-nrpe configuration files.

Author: Marc Fournier <marc.fournier@camptocamp.com>

About: License
  This file is licensed under the LGPL v2+, like the rest of Augeas.
*)

module Nrpe =
  autoload xfm


let eol = Util.eol
let eq = Sep.equal

(* View: word *)
let word = /[^=\n\t ]+/

(* View: item_re *)
let item_re = /[^#=\n\t\/ ]+/ - (/command\[[^]\/\n]+\]/ | "include" | "include_dir")

(* View: command
    nrpe.cfg usually has many entries defining commands to run

    > command[check_foo]=/path/to/nagios/plugin -w 123 -c 456
    > command[check_bar]=/path/to/another/nagios/plugin --option
*)
let command =
  let obrkt = del /\[/ "[" in
  let cbrkt = del /\]/ "]" in
    [ key "command" .
    [ obrkt . key /[^]\/\n]+/ . cbrkt . eq
            . store /[^\n]+/ . del /\n/ "\n" ]
    ]


(* View: item
     regular entries

     > allow_bash_command_substitution=0
*)
let item = [ key item_re . eq . store word . eol ]

(* View: include
    An include entry.

    nrpe.cfg can include more than one file or directory of files

    > include=/path/to/file1.cfg
    > include=/path/to/file2.cfg
*)
let include = [ key "include" .
  [ label "file" . eq . store word . eol ]
]

(* View: include_dir
    > include_dir=/path/to/dir/
*)
let include_dir = [ key "include_dir" .
  [ label "dir" . eq . store word . eol ]
]


(* View: comment
    Nrpe comments must start at beginning of line *)
let comment = Util.comment_generic /#[ \t]*/ "# "

(* blank lines and empty comments *)
let empty = Util.empty

(* View: lns
    The Nrpe lens *)
let lns = ( command | include | include_dir | item | comment | empty ) *

(* View: filter
    File filter *)
let filter = incl "/etc/nrpe.cfg" .
             incl "/etc/nagios/nrpe.cfg"

let xfm = transform lns (filter)

