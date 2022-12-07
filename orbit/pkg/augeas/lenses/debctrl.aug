(*
Module: debctrl
  Parses ./debian/control

Author:
        Dominique Dumont domi.dumont@free.fr or dominique.dumont@hp.com

About: Reference
  http://augeas.net/page/Create_a_lens_from_bottom_to_top
  http://www.debian.org/doc/debian-policy/ch-controlfields.html

About: License
  This file is licensed under the LGPL v2+.

About: Lens Usage
  Since control file is not a system configuration file, you will have
  to use augtool -r option to point to 'debian' directory.

  Run augtool:
  $ augtool -r debian

  Sample usage of this lens in augtool:

    * Get the value stored in control file
      > print /files/control
      ...

  Saving your file:

      > save


*)

module Debctrl =
  autoload xfm

let eol = Util.eol
let del_ws_spc = del /[\t ]*/ " "
let hardeol = del /\n/ "\n"
let del_opt_ws = del /[\t ]*/ ""
let colon = del /:[ \t]*/ ": "

let simple_entry (k:regexp) =
   let value =  store /[^ \t][^\n]+/ in
   [ key k . colon . value . hardeol ]

let cont_line = del /\n[ \t]+/ "\n "
let comma     = del  /,[ \t]*/  ", "

let sep_comma_with_nl = del /[ \t\n]*,[ \t\n]*/ ",\n "
 (*= del_opt_ws . cont_line* . comma . cont_line**)

let email =  store ( /([A-Za-z]+ )+<[^\n>]+>/ |  /[^\n,\t<> ]+/ )

let multi_line_array_entry (k:regexp) (v:lens) =
    [ key k . colon . [ counter "array" . seq "array" .  v ] .
      [ seq "array" . sep_comma_with_nl . v ]* . hardeol ]

(* dependency stuff *)

let version_depends =
    [ label "version"
     . [   del / *\( */ " ( " . label "relation" . store /[<>=]+/ ]
     . [   del_ws_spc . label "number"
           . store ( /[a-zA-Z0-9_.-]+/ | /\$\{[a-zA-Z0-9:]+\}/ )
         . del / *\)/ " )" ]
    ]

let arch_depends =
    [ label "arch"
    . [  del / *\[ */ " [ " . label "prefix" . store /!?/ ]
    . [ label "name" . store /[a-zA-Z0-9_.-]+/ . del / *\]/ " ]" ] ]


let package_depends
  =  [ key ( /[a-zA-Z0-9_-]+/ | /\$\{[a-zA-Z0-9:]+\}/ )
        . ( version_depends | arch_depends ) * ]


let dependency = [ label "or" . package_depends ]
               . [ label "or" . del / *\| */ " | "
                   . package_depends ] *

let dependency_list (field:regexp) =
    [ key field . colon . [ label "and" .  dependency ]
      . [ label "and" . sep_comma_with_nl . dependency ]*
      . eol ]

(* source package *)
let uploaders  =
    multi_line_array_entry /Uploaders/ email

let simple_src_keyword = "Source" | "Section" | "Priority"
    | "Standards\-Version" | "Homepage" | /Vcs\-Svn/ | /Vcs\-Browser/
    | "Maintainer" | "DM-Upload-Allowed" | /XS?-Python-Version/
let depend_src_keywords = /Build\-Depends/ | /Build\-Depends\-Indep/

let src_entries = (   simple_entry simple_src_keyword
                    | uploaders
                    | dependency_list depend_src_keywords ) *


(* package paragraph *)
let multi_line_entry (k:string) =
     let line = /.*[^ \t\n].*/ in
      [ label k .  del / / " " .  store line . hardeol ] *


let description
  = [ key "Description" . colon
     . [ label "summary" . store /[a-zA-Z][^\n]+/ . hardeol ]
     . multi_line_entry "text" ]


(* binary package *)
let simple_bin_keywords = "Package" | "Architecture" |  "Section"
    | "Priority" | "Essential" | "Homepage" | "XB-Python-Version"
let depend_bin_keywords = "Depends" | "Recommends" | "Suggests" | "Provides"

let bin_entries = ( simple_entry simple_bin_keywords
                  | dependency_list depend_bin_keywords
                  ) + . description

(* The whole stuff *)
let lns =  [ label "srcpkg" .  src_entries  ]
        .  [ label "binpkg" . hardeol+ . bin_entries ]+
        . eol*

(* lens must be used with AUG_ROOT set to debian package source directory *)
let xfm = transform lns (incl "/control")
