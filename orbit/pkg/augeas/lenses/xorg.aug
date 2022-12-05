(*
Module: Xorg
 Parses /etc/X11/xorg.conf

Authors: Raphael Pinson <raphink@gmail.com>
         Matthew Booth <mbooth@redhat.com>

About: Reference
 This lens tries to keep as close as possible to `man xorg.conf` where
 possible.

The definitions from `man xorg.conf` are put as commentaries for reference
throughout the file. More information can be found in the manual.

About: License
  This file is licensed under the LGPLv2+, like the rest of Augeas.

About: Lens Usage
  Sample usage of this lens in augtool

    * Get the identifier of the devices with a "Clone" option:
      > match "/files/etc/X11/xorg.conf/Device[Option = 'Clone']/Identifier"

About: Configuration files
  This lens applies to /etc/X11/xorg.conf. See <filter>.
*)

module Xorg =
  autoload xfm

(************************************************************************
 * Group:                 USEFUL PRIMITIVES
 *************************************************************************)

(* Group: Generic primitives *)

(* Variable: eol *)
let eol     = Util.eol

(* Variable: to_eol
 * Match everything from here to eol, cropping whitespace at both ends
 *)
let to_eol  = /[^ \t\n](.*[^ \t\n])?/

(* Variable: indent *)
let indent  = Util.indent

(* Variable: comment *)
let comment = Util.comment

(* Variable: empty *)
let empty   = Util.empty


(* Group: Separators *)

(* Variable: sep_spc *)
let sep_spc = Util.del_ws_spc

(* Variable: sep_dquote *)
let sep_dquote  = Util.del_str "\""


(* Group: Fields and values *)

(* Variable: entries_re
 * This is a list of all patterns which have specific handlers, and should
 * therefore not be matched by the generic handler
 *)
let entries_re  = /([oO]ption|[sS]creen|[iI]nput[dD]evice|[dD]river|[sS]ub[sS]ection|[dD]isplay|[iI]dentifier|[vV]ideo[rR]am|[dD]efault[dD]epth|[dD]evice)/

(* Variable: generic_entry_re *)
let generic_entry_re = /[^# \t\n\/]+/ - entries_re

(* Variable: quoted_non_empty_string_val *)
let quoted_non_empty_string_val = del "\"" "\"" . store /[^"\n]+/
                                  . del "\"" "\""
                                              (* " relax, emacs *)

(* Variable: quoted_string_val *)
let quoted_string_val = del "\"" "\"" . store /[^"\n]*/ . del "\"" "\""
                                              (* " relax, emacs *)

(* Variable: int *)
let int = /[0-9]+/


(************************************************************************
 * Group:                          ENTRIES AND OPTIONS
 *************************************************************************)


(* View: entry_int
 * This matches an entry which takes a single integer for an argument
 *)
let entry_int (canon:string) (re:regexp) =
        [ indent . del re canon . label canon . sep_spc . store int . eol ]

(* View: entry_rgb
 * This matches an entry which takes 3 integers as arguments representing red,
 * green and blue components
 *)
let entry_rgb (canon:string) (re:regexp) =
        [ indent . del re canon . label canon
          . [ label "red"   . sep_spc . store int ]
          . [ label "green" . sep_spc . store int ]
          . [ label "blue"  . sep_spc . store int ]
          . eol ]

(* View: entry_xy
 * This matches an entry which takes 2 integers as arguments representing X and
 * Y coordinates
 *)
let entry_xy (canon:string) (re:regexp) =
        [ indent . del re canon . label canon
          . [ label "x" . sep_spc . store int ]
          . [ label "y" . sep_spc . store int ]
          . eol ]

(* View: entry_str
 * This matches an entry which takes a single quoted string
 *)
let entry_str (canon:string) (re:regexp) =
        [ indent . del re canon . label canon
          . sep_spc . quoted_non_empty_string_val . eol ]

(* View: entry_generic
 * An entry without a specific handler. Store everything after the keyword,
 * cropping whitespace at both ends.
 *)
let entry_generic  = [ indent . key generic_entry_re
                       . sep_spc . store to_eol . eol ]

(* View: option *)
let option = [ indent . del /[oO]ption/ "Option" . label "Option" . sep_spc
               . quoted_non_empty_string_val
               . [ label "value" . sep_spc . quoted_string_val ]*
               . eol ]

(* View: screen
 * The Screen entry of ServerLayout
 *)
let screen = [ indent . del /[sS]creen/ "Screen" . label "Screen"
               . [ sep_spc . label "num" . store int ]?
               . ( sep_spc . quoted_non_empty_string_val
               . [ sep_spc . label "position" . store to_eol ]? )?
               . eol ]

(* View: input_device *)
let input_device = [ indent . del /[iI]nput[dD]evice/ "InputDevice"
                     . label "InputDevice" . sep_spc
		     . quoted_non_empty_string_val
                     . [ label "option" . sep_spc
		         . quoted_non_empty_string_val ]*
                     . eol ]

(* View: driver *)
let driver = entry_str "Driver" /[dD]river/

(* View: identifier *)
let identifier = entry_str "Identifier" /[iI]dentifier/

(* View: videoram *)
let videoram = entry_int "VideoRam" /[vV]ideo[rR]am/

(* View: default_depth *)
let default_depth = entry_int "DefaultDepth" /[dD]efault[dD]epth/

(* View: device *)
let device = entry_str "Device" /[dD]evice/

(************************************************************************
 * Group:                          DISPLAY SUBSECTION
 *************************************************************************)


(* View: display_modes *)
let display_modes = [ indent . del /[mM]odes/ "Modes" . label "Modes"
                      . [ label "mode" . sep_spc
		          . quoted_non_empty_string_val ]+
                      . eol ]

(*************************************************************************
 * View: display_entry
 *   Known values for entries in the Display subsection
 *
 *   Definition:
 *     > Depth    depth
 *     > FbBpp    bpp
 *     > Weight   red-weight green-weight blue-weight
 *     > Virtual  xdim ydim
 *     > ViewPort x0 y0
 *     > Modes    "mode-name" ...
 *     > Visual   "visual-name"
 *     > Black    red green blue
 *     > White    red green blue
 *     > Options
 *)

let display_entry = entry_int "Depth"    /[dD]epth/ |
                    entry_int "FbBpp"    /[fF]b[bB]pp/ |
                    entry_rgb "Weight"   /[wW]eight/ |
                    entry_xy  "Virtual"  /[vV]irtual/ |
                    entry_xy  "ViewPort" /[vV]iew[pP]ort/ |
                    display_modes |
                    entry_str "Visual"   /[vV]isual/ |
                    entry_rgb "Black"    /[bB]lack/ |
                    entry_rgb "White"    /[wW]hite/ |
                    entry_str "Options"  /[oO]ptions/ |
                    empty |
                    comment

(* View: display *)
let display = [ indent . del "SubSection" "SubSection" . sep_spc
                       . sep_dquote . key "Display" . sep_dquote
                       . eol
                       . display_entry*
                       . indent . del "EndSubSection" "EndSubSection" . eol ]

(************************************************************************
 * Group:                          EXTMOD SUBSECTION
 *************************************************************************)

let extmod_entry =  entry_str "Option"  /[oO]ption/ |
                    empty |
                    comment

let extmod = [ indent . del "SubSection" "SubSection" . sep_spc
                       . sep_dquote . key "extmod" . sep_dquote
                       . eol
                       . extmod_entry*
                       . indent . del "EndSubSection" "EndSubSection" . eol ]

(************************************************************************
 * Group:                       SECTIONS
 *************************************************************************)


(************************************************************************
 * Variable: section_re
 *   Known values for Section names
 *
 *   Definition:
 *     >   The section names are:
 *     >
 *     >   Files          File pathnames
 *     >   ServerFlags    Server flags
 *     >   Module         Dynamic module loading
 *     >   Extensions     Extension Enabling
 *     >   InputDevice    Input device description
 *     >   InputClass     Input Class description
 *     >   Device         Graphics device description
 *     >   VideoAdaptor   Xv video adaptor description
 *     >   Monitor        Monitor description
 *     >   Modes          Video modes descriptions
 *     >   Screen         Screen configuration
 *     >   ServerLayout   Overall layout
 *     >   DRI            DRI-specific configuration
 *     >   Vendor         Vendor-specific configuration
 *************************************************************************)
let section_re = /(Extensions|Files|ServerFlags|Module|InputDevice|InputClass|Device|VideoAdaptor|Monitor|Modes|Screen|ServerLayout|DRI|Vendor)/


(************************************************************************
 * Variable: secton_re_obsolete
 *   The  following obsolete section names are still recognised for
 *   compatibility purposes.  In new config files, the InputDevice
 *   section should be used instead.
 *
 *   Definition:
 *     >  Keyboard       Keyboard configuration
 *     >  Pointer        Pointer/mouse configuration
 *************************************************************************)
let section_re_obsolete = /(Keyboard|Pointer)/

(* View: section_entry *)
let section_entry = option |
                    screen |
                    display |
                    extmod |
                    input_device |
                    driver |
                    identifier |
                    videoram |
                    default_depth |
                    device |
                    entry_generic |
                    empty | comment

(************************************************************************
 * View: section
 *   A section in xorg.conf
 *
 *   Definition:
 *     > Section  "SectionName"
 *     >    SectionEntry
 *     >    ...
 *     > EndSection
 *************************************************************************)
let section = [ indent . del "Section" "Section"
                       . sep_spc . sep_dquote
                       . key (section_re|section_re_obsolete) . sep_dquote
                       . eol
                .  section_entry*
                . indent . del "EndSection" "EndSection" . eol ]

(*
 * View: lns
 *   The xorg.conf lens
 *)
let lns = ( empty | comment | section )*


(* Variable: filter *)
let filter = incl "/etc/X11/xorg.conf"
           . incl "/etc/X11/xorg.conf.d/*.conf"
           . Util.stdexcl

let xfm = transform lns filter
