(*
Module: Mke2fs
  Parses /etc/mke2fs.conf

Author: Raphael Pinson <raphink@gmail.com>

About: Reference
  This lens tries to keep as close as possible to `man 5 mke2fs.conf` where possible.

About: License
   This file is licenced under the LGPL v2+, like the rest of Augeas.

About: Lens Usage
   To be documented

About: Configuration files
   This lens applies to /etc/mke2fs.conf. See <filter>.
*)


module Mke2fs =
  autoload xfm

(************************************************************************
 * Group:                 USEFUL PRIMITIVES
 *************************************************************************)

(* View: comment *)
let comment = IniFile.comment IniFile.comment_re IniFile.comment_default

(* View: sep *)
let sep = IniFile.sep /=[ \t]*/ "="

(* View: empty *)
let empty = IniFile.empty

(* View: boolean
    The configuration parser of e2fsprogs recognizes different values
    for booleans, so list all the recognized values *)
let boolean = ("y"|"yes"|"true"|"t"|"1"|"on"|"n"|"no"|"false"|"nil"|"0"|"off")

(* View: fspath *)
let fspath = /[^ \t\n"]+/


(************************************************************************
 * Group:                 RECORD TYPES
 *************************************************************************)


(* View: entry
    A generic entry for lens lns *)
let entry (kw:regexp) (lns:lens) = Build.key_value_line kw sep lns


(* View: list_sto
    A list of values with given lens *)
let list_sto (kw:regexp) (lns:lens) =
  entry kw (Quote.do_dquote_opt_nil (Build.opt_list [lns] Sep.comma))

(* View: entry_sto
    Store a regexp as entry value *)
let entry_sto (kw:regexp) (val:regexp) =
  entry kw (Quote.do_dquote_opt_nil (store val))
  | entry kw (Util.del_str "\"\"")


(************************************************************************
 * Group:                 COMMON ENTRIES
 *************************************************************************)

(* View: common_entries_list
    Entries with a list value *)
let common_entries_list = ("base_features"|"default_features"|"default_mntopts")

(* View: common_entries_int
    Entries with an integer value *)
let common_entries_int = ("cluster_size"|"flex_bg_size"|"force_undo"
                         |"inode_ratio"|"inode_size"|"num_backup_sb")

(* View: common_entries_bool
    Entries with a boolean value *)
let common_entries_bool = ("auto_64-bit_support"|"discard"
                          |"enable_periodic_fsck"|"lazy_itable_init"
                          |"lazy_journal_init"|"packed_meta_blocks")

(* View: common_entries_string
    Entries with a string value *)
let common_entries_string = ("encoding"|"journal_location")

(* View: common_entries_double
    Entries with a double value *)
let common_entries_double = ("reserved_ratio")

(* View: common_entry
     Entries shared between <defaults> and <fs_types> sections *)
let common_entry   = list_sto common_entries_list (key Rx.word)
                   | entry_sto common_entries_int Rx.integer
                   | entry_sto common_entries_bool boolean
                   | entry_sto common_entries_string Rx.word
                   | entry_sto common_entries_double Rx.decimal
                   | entry_sto "blocksize" ("-"? . Rx.integer)
                   | entry_sto "hash_alg" ("legacy"|"half_md4"|"tea")
                   | entry_sto "errors" ("continue"|"remount-ro"|"panic")
                   | list_sto "features"
                        ([del /\^/ "^" . label "disable"]?
                                           . key Rx.word)
                   | list_sto "options"
                        (key Rx.word . Util.del_str "="
                       . store Rx.word)

(************************************************************************
 * Group:                 DEFAULTS SECTION
 *************************************************************************)

(* View: defaults_entry
    Possible entries under the <defaults> section *)
let defaults_entry = entry_sto "fs_type" Rx.word
                   | entry_sto "undo_dir" fspath
                   
(* View: defaults_title
    Title for the <defaults> section *)
let defaults_title  = IniFile.title "defaults"

(* View: defaults
    A defaults section *)
let defaults = IniFile.record defaults_title
                  ((Util.indent . (defaults_entry|common_entry)) | comment)


(************************************************************************
 * Group:                 FS_TYPES SECTION
 *************************************************************************)

(* View: fs_types_record
     Fs group records under the <fs_types> section *)
let fs_types_record = [ label "filesystem"
                     . Util.indent . store Rx.word
                     . del /[ \t]*=[ \t]*\{[ \t]*\n/ " = {\n"
                     . ((Util.indent . common_entry) | empty | comment)*
                     . del /[ \t]*\}[ \t]*\n/ " }\n" ]

(* View: fs_types_title
    Title for the <fs_types> section *)
let fs_types_title = IniFile.title "fs_types"

(* View: fs_types
    A fs_types section *)
let fs_types = IniFile.record fs_types_title
                  (fs_types_record | comment)


(************************************************************************
 * Group:                 OPTIONS SECTION
 *************************************************************************)

(* View: options_entries_int
    Entries with an integer value *)
let options_entries_int = ("proceed_delay"|"sync_kludge")

(* View: options_entries_bool
    Entries with a boolean value *)
let options_entries_bool = ("old_bitmaps")

(* View: options_entry
    Possible entries under the <options> section *)
let options_entry = entry_sto options_entries_int Rx.integer
                  | entry_sto options_entries_bool boolean

(* View: defaults_title
    Title for the <options> section *)
let options_title  = IniFile.title "options"

(* View: options
    A options section *)
let options = IniFile.record options_title
                  ((Util.indent . options_entry) | comment)


(************************************************************************
 * Group:                 LENS AND FILTER
 *************************************************************************)

(* View: lns
     The mke2fs lens
*)
let lns = (empty|comment)* . (defaults|fs_types|options)*

(* Variable: filter *)
let filter = incl "/etc/mke2fs.conf"

let xfm = transform lns filter


