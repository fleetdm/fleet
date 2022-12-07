(*
Module: Kdump
  Parses /etc/kdump.conf

Author: Roman Rakus <rrakus@redhat.com>

About: References
  manual page kdump.conf(5)

About: License
   This file is licenced under the LGPL v2+, like the rest of Augeas.

About: Configuration files
   This lens applies to /etc/kdump.conf. See <filter>.
*)

module Kdump =
  autoload xfm

(************************************************************************
 * Group:                 USEFUL PRIMITIVES
 *************************************************************************)

let empty = Util.empty
let comment = Util.comment
let value_to_eol = store /[^ \t\n#][^\n#]*[^ \t\n#]|[^ \t\n#]/
let int_to_eol = store Rx.integer
let yn_to_eol = store ("yes" | "no")
let delimiter = Util.del_ws_spc
let eol = Util.eol
let value_to_spc = store Rx.neg1
let key_to_space = key /[A-Za-z0-9_.\$-]+/
let eq = Sep.equal

(************************************************************************
 * Group:                 ENTRY TYPES
 *************************************************************************)

let list (kw:string) = counter kw
                     . Build.key_value_line_comment kw delimiter
                         (Build.opt_list [ seq kw . value_to_spc ] delimiter)
                         comment

let mdl_key_value = [ delimiter . key_to_space . ( eq . value_to_spc)? ]
let mdl_options = [ key_to_space . mdl_key_value+ ]
let mod_options = [ key "options" . delimiter . mdl_options . (comment|eol) ]

(************************************************************************
 * Group:                 ENTRIES
 *************************************************************************)

(* Got from mount(8) *)
let fs_types = "adfs" | "affs" | "autofs" | "cifs" | "coda" | "coherent"
             | "cramfs" | "debugfs" | "devpts" | "efs" | "ext" | "ext2"
             | "ext3" | "ext4" | "hfs" | "hfsplus" | "hpfs" | "iso9660"
             | "jfs" | "minix" | "msdos" | "ncpfs" | "nfs" | "nfs4" | "ntfs"
             | "proc" | "qnx4" | "ramfs" | "reiserfs" | "romfs" | "squashfs"
             | "smbfs" | "sysv" | "tmpfs" | "ubifs" | "udf" | "ufs" | "umsdos"
             | "usbfs" | "vfat" | "xenix" | "xfs" | "xiafs"

let simple_kws = "raw" | "net" | "path" | "core_collector" | "kdump_post"
               | "kdump_pre" | "default" | "ssh" | "sshkey" | "dracut_args"
               | "fence_kdump_args"

let int_kws = "force_rebuild" | "override_resettable" | "debug_mem_level"
            | "link_delay" | "disk_timeout"

let yn_kws = "auto_reset_crashkernel"

let option = Build.key_value_line_comment ( simple_kws | fs_types )
                                          delimiter value_to_eol comment
           | Build.key_value_line_comment int_kws delimiter int_to_eol comment
           | Build.key_value_line_comment yn_kws delimiter yn_to_eol comment
           | list "extra_bins"
           | list "extra_modules"
           | list "blacklist"
           | list "fence_kdump_nodes"
           | mod_options

(* View: lns
   The options lens
*)
let lns = ( empty | comment | option )*

let filter = incl "/etc/kdump.conf"

let xfm = transform lns filter
