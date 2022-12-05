(*
Module: BootConf
    Parses (Open)BSD-stype /etc/boot.conf

Author: Jasper Lievisse Adriaanse <jasper@jasper.la>

About: Reference
    This lens is used to parse the second-stage bootstrap configuration
    file, /etc/boot.conf as found on OpenBSD. The format is largely MI,
    with MD parts included:
    http://www.openbsd.org/cgi-bin/man.cgi?query=boot.conf&arch=i386

About: Usage Example
    To be documented

About: License
    This file is licensed under the LGPL v2+, like the rest of Augeas.

About: Configuration files
  This lens applies to /etc/boot.conf.
See <filter>.
*)

module BootConf =
autoload xfm

(************************************************************************
 * Utility variables/functions
 ************************************************************************)

(* View: comment *)
let comment = Util.comment
(* View:  empty *)
let empty   = Util.empty
(* View: eol *)
let eol     = Util.eol
(* View: fspath *)
let fspath  = Rx.fspath
(* View: space *)
let space   = Sep.space
(* View: word *)
let word    = Rx.word

(************************************************************************
 * View: key_opt_value_line
 *   A subnode with a keyword, an optional part consisting of a separator
 *   and a storing lens, and an end of line
 *
 *   Parameters:
 *     kw:regexp - the pattern to match as key
 *     sto:lens  - the storing lens
 ************************************************************************)
let key_opt_value_line (kw:regexp) (sto:lens) =
    [ key kw . (space . sto)? . eol ]

(************************************************************************
 * Commands
 ************************************************************************)

(* View: single_command
   single command such as 'help' or 'time' *)
let single_command =
    let line_re = /help|time|reboot/
      in [ Util.indent . key line_re . eol ]

(* View: ls
   ls [directory] *)
let ls = Build.key_value_line
               "ls" space (store fspath)

let set_cmd = "addr"
            | "debug"
            | "device"
            | "howto"
            | "image"
            | "timeout"
            | "tty"

(* View: set
   set [varname [value]] *)
let set = Build.key_value
               "set" space
	       (key_opt_value_line set_cmd (store Rx.space_in))

(* View: stty
  stty [device [speed]] *)
let stty =
    let device = [ label "device" . store fspath ]
      in let speed = [ label "speed" . store Rx.integer ]
        in key_opt_value_line "stty" (device . (space . speed)?)

(* View: echo
   echo [args] *)
let echo = Build.key_value_line
                 "echo" space (store word)

(* View: boot
   boot [image [-acds]]
   XXX: the last arguments are not always needed, so make them optional *)
let boot =
    let image = [ label "image" . store fspath ]
      in let arg = [ label "arg" . store word ]
        in Build.key_value_line "boot" space (image . space . arg)

(* View: machine
   machine [command] *)
let machine =
      let machine_entry = Build.key_value ("comaddr"|"memory") 
                                          space (store word)
                        | Build.flag ("diskinfo"|"regs")
            in Build.key_value_line
                    "machine" space
                    (Build.opt_list
                           machine_entry
                           space)

(************************************************************************
 * Lens
 ************************************************************************)

(* View: command *)
let command = boot | echo | ls | machine | set | stty

(* View: lns *)
let lns = ( empty | comment | command | single_command )*

(* Variable: filter *)
let filter = (incl "/etc/boot.conf")

let xfm = transform lns filter
