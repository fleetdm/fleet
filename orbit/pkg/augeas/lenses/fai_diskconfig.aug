(*
Module: FAI_DiskConfig
 Parses disk_config files for FAI

Author: Raphael Pinson <raphink@gmail.com>

About: Reference
 This lens tries to keep as close as possible to the FAI wiki where possible:
 http://wiki.fai-project.org/wiki/Setup-storage#New_configuration_file_syntax

About: License
  This file is licensed under the LGPL v2+, like the rest of Augeas.

About: Examples
   The <Test_FAI_DiskConfig> file contains various examples and tests.
*)

module FAI_DiskConfig =

(* autoload xfm *)

(************************************************************************
 * Group:                 USEFUL PRIMITIVES
 *************************************************************************)

(* Group: Generic primitives *)
(* Variable: eol *)
let eol = Util.eol

(* Variable: space *)
let space = Sep.space

(* Variable: empty *)
let empty = Util.empty

(* Variable: comment *)
let comment = Util.comment

(* Variable: tag
   A generic tag beginning with a colon *)
let tag (re:regexp) = [ Util.del_str ":" . key re ]

(* Variable: generic_opt
   A generic key/value option *)
let generic_opt (type:string) (kw:regexp) =
   [ key type . Util.del_str ":" . store kw ]

(* Variable: generic_opt_list
   A generic key/list option *)
let generic_opt_list (type:string) (kw:regexp) =
   [ key type . Util.del_str ":" . counter "locallist"
   . Build.opt_list [seq "locallist" . store kw] Sep.comma ]


(************************************************************************
 * Group:                      RECORDS
 *************************************************************************)


(* Group: volume *)

(* Variable: mountpoint_kw *)
let mountpoint_kw = "-" (* do not mount *)
         | "swap"       (* swap space *)
         (* fully qualified path; if :encrypt is given, the partition
          * will be encrypted, the key is generated automatically *)
         | /\/[^: \t\n]*/

(* Variable: encrypt
   encrypt tag *)
let encrypt = tag "encrypt"

(* Variable: mountpoint *)
let mountpoint = [ label "mountpoint" . store mountpoint_kw
                 (* encrypt is only for the fspath, but we parse it anyway *)
                 . encrypt?]

(* Variable: resize
   resize tag *)
let resize = tag "resize"

(* Variable: size_kw
   Regexps for size *)
let size_kw = /[0-9]+[kMGTP%]?(-([0-9]+[kMGTP%]?)?)?/
            | /-[0-9]+[kMGTP%]?/

(* Variable: size *)
let size = [ label "size" . store size_kw . resize? ]

(* Variable: filesystem_kw
   Regexps for filesystem *)
let filesystem_kw = "-"
         | "swap"
         (* NOTE: Restraining this regexp would improve perfs *)
         | (Rx.no_spaces - ("-" | "swap")) (* mkfs.xxx must exist *)

(* Variable: filesystem *)
let filesystem = [ label "filesystem" . store filesystem_kw ]


(* Variable: mount_option_value *)
let mount_option_value = [ label "value" . Util.del_str "="
                         . store /[^,= \t\n]+/ ]

(* Variable: mount_option
   Counting options *)
let mount_option = [ seq "mount_option"
                   . store /[^,= \t\n]+/
                   . mount_option_value? ]

(* Variable: mount_options
   An array of <mount_option>s *)
let mount_options = [ label "mount_options"
                    . counter "mount_option"
                    . Build.opt_list mount_option Sep.comma ]

(* Variable: fs_option *)
let fs_option =
     [ key /createopts|tuneopts/
     . Util.del_str "=\"" . store /[^"\n]*/ . Util.del_str "\"" ]

(* Variable: fs_options
   An array of <fs_option>s *)
let fs_options =
     (* options to append to mkfs.xxx and to the filesystem-specific
      * tuning tool *)
     [ label "fs_options" . Build.opt_list fs_option Sep.space ]

(* Variable: volume_full *)
let volume_full (type:lens) (third_field:lens) =
           [ type . space
           . mountpoint .space
           (* The third field changes depending on types *)
           . third_field . space
           . filesystem . space
           . mount_options
           . (space . fs_options)?
           . eol ]

(* Variable: name
   LVM volume group name *)
let name = [ label "name" . store /[^\/ \t\n]+/ ]

(* Variable: partition
   An optional partition number for <disk> *)
let partition = [ label "partition" . Util.del_str "." . store /[0-9]+/ ]

(* Variable: disk *)
let disk = [ label "disk" . store /[^., \t\n]+/ . partition? ]

(* Variable: vg_option
   An option for <volume_vg> *)
let vg_option =
     [ key "pvcreateopts"
     . Util.del_str "=\"" . store /[^"\n]*/ . Util.del_str "\"" ]

(* Variable: volume_vg *)
let volume_vg = [ key "vg"
                . space . name
                . space . disk
                . (space . vg_option)?
                . eol ]

(* Variable: spare_missing *)
let spare_missing = tag /spare|missing/

(* Variable: disk_with_opt
   A <disk> with a spare/missing option for raids *)
let disk_with_opt = [ label "disk" . store /[^:., \t\n]+/ . partition?
                    . spare_missing* ]

(* Variable: disk_list
   A list of <disk_with_opt>s *)
let disk_list = Build.opt_list disk_with_opt Sep.comma

(* Variable: type_label_lv *)
let type_label_lv = label "lv"
                    . [ label "vg" . store (/[^# \t\n-]+/ - "raw") ]
                    . Util.del_str "-"
                    . [ label "name" . store /[^ \t\n]+/ ]

(* Variable: volume_tmpfs *)
let volume_tmpfs =
           [ key "tmpfs" . space
           . mountpoint .space
           . size . space
           . mount_options
           . (space . fs_options)?
           . eol ]

(* Variable: volume_lvm *)
let volume_lvm = volume_full type_label_lv size  (* lvm logical volume: vg name and lv name *)
               | volume_vg

(* Variable: volume_raid *)
let volume_raid = volume_full (key /raid[0156]/) disk_list  (* raid level *)

(* Variable: device *)
let device = [ label "device" . store Rx.fspath ]

(* Variable: volume_cryptsetup *)
let volume_cryptsetup = volume_full (key ("swap"|"tmp"|"luks")) device

(* Variable: volume *)
let volume = volume_full (key "primary") size     (* for physical disks only *)
           | volume_full (key "logical") size     (* for physical disks only *)
           | volume_full (key "raw-disk") size

(* Variable: volume_or_comment
   A succesion of <volume>s and <comment>s *)
let volume_or_comment (vol:lens) =
      (vol|empty|comment)* . vol

(* Variable: disk_config_entry *)
let disk_config_entry (kw:regexp) (opt:lens) (vol:lens) =
                  [ key "disk_config" . space . store kw
                  . (space . opt)* . eol
                  . (volume_or_comment vol)? ]

(* Variable: lvmoption *)
let lvmoption =
     (* preserve partitions -- always *)
      generic_opt "preserve_always" /[^\/, \t\n-]+-[^\/, \t\n-]+(,[^\/, \t\n-]+-[^\/, \t\n-]+)*/
     (* preserve partitions -- unless the system is installed
      * for the first time *)
   | generic_opt "preserve_reinstall" /[^\/, \t\n-]+-[^\/, \t\n-]+(,[^\/, \t\n-]+-[^\/, \t\n-]+)*/
     (* attempt to resize partitions *)
   | generic_opt "resize" /[^\/, \t\n-]+-[^\/, \t\n-]+(,[^\/, \t\n-]+-[^\/, \t\n-]+)*/
     (* when creating the fstab, the key used for defining the device
      * may be the device (/dev/xxx), a label given using -L, or the uuid *)
   | generic_opt "fstabkey" /device|label|uuid/

(* Variable: raidoption *)
let raidoption =
     (* preserve partitions -- always *)
     generic_opt_list "preserve_always" (Rx.integer | "all")
     (* preserve partitions -- unless the system is installed
      * for the first time *)
   | generic_opt_list "preserve_reinstall" Rx.integer
     (* when creating the fstab, the key used for defining the device
      * may be the device (/dev/xxx), a label given using -L, or the uuid *)
   | generic_opt "fstabkey" /device|label|uuid/

(* Variable: option *)
let option =
     (* preserve partitions -- always *)
     generic_opt_list "preserve_always" (Rx.integer | "all")
     (* preserve partitions -- unless the system is installed
        for the first time *)
   | generic_opt_list "preserve_reinstall" Rx.integer
     (* attempt to resize partitions *)
   | generic_opt_list "resize" Rx.integer
     (* write a disklabel - default is msdos *)
   | generic_opt "disklabel" /msdos|gpt/
     (* mark a partition bootable, default is / *)
   | generic_opt "bootable" Rx.integer
     (* do not assume the disk to be a physical device, use with xen *)
   | [ key "virtual" ]
     (* when creating the fstab, the key used for defining the device
      * may be the device (/dev/xxx), a label given using -L, or the uuid *)
   | generic_opt "fstabkey" /device|label|uuid/
   | generic_opt_list "always_format" Rx.integer
   | generic_opt "sameas" Rx.fspath

let cryptoption =
     [ key "randinit" ]

(* Variable: disk_config *)
let disk_config =
    let excludes = "lvm" | "raid" | "end" | /disk[0-9]+/
                 | "cryptsetup" | "tmpfs" in
    let other_label = Rx.fspath - excludes in
                  disk_config_entry "lvm" lvmoption volume_lvm
                | disk_config_entry "raid" raidoption volume_raid
                | disk_config_entry "tmpfs" option volume_tmpfs
                | disk_config_entry "end" option volume (* there shouldn't be an option here *)
                | disk_config_entry /disk[0-9]+/ option volume
                | disk_config_entry "cryptsetup" cryptoption volume_cryptsetup
                | disk_config_entry other_label option volume

(* Variable: lns
   The disk_config lens *)
let lns = (disk_config|comment|empty)*


(* let xfm = transform lns Util.stdexcl *)
