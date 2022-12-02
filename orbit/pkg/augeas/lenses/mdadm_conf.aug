(******************************************************************************
Mdadm_conf module for Augeas

Author: Matthew Booth <mbooth@redhat.com>

Copyright (C):
    2011 Red Hat Inc.

Reference:
    mdadm(5)
    config.c and policy.c from mdadm-3.2.2

License:
    This file is licensed under the LGPL v2+.

This is a lens for /etc/mdadm.conf. It aims to parse every valid configuration
file as of version 3.2.2, and many invalid ones too. This last point is a
feature, not a bug! madm will generate warnings for invalid configuration which
do not prevent correct operation of the tool. Wherever possible, we try to
allow for this behaviour.

Keywords in mdadm.conf are matched with a case-insensitive prefix match of at
least 3 characters. Keys in key/value pairs are also matched case-insensitively,
but require a full match. The exception is POLICY and PART-POLICY, where keys
are matched case-sensitively.

N.B. We can't use case-insensitive regular expressions in most places due to bug
#147.
*******************************************************************************)

module Mdadm_conf =

   autoload xfm


(******************************************************************************
 * PRIMITIVES
 ******************************************************************************)

let eol             = Util.comment_or_eol
let comment         = Util.comment
let empty           = Util.empty
let value           = /[^ \t\n#]+/
let value_no_eq     = /[^ \t\n#=]+/
let value_no_eq_sl  = /[^ \t\n#=\/]+/

let continuation = /\n[ \t]+/
let space = /[ \t]+/
let value_sep = ( del ( continuation | space . continuation? ) " "
                  | comment . del space " " )

(* We parse specific keys rather than having a catch-all owing to the varying
case of the syntax. This means the user can rely on 'array/uuid' rather than
additionally testing for 'array/UUID'.

It would be good to have an additional catchall, but I haven't been able to make
that work.
*)
let keyvalue (r:regexp) (lc:string) (uc:string) =
    [ del ( r . /=/ ) ( uc . "=" ) . label lc . store value ]

let simplevalue (r:regexp) (lc:string) (uc:string) =
    [ del r uc . label lc
      . ( value_sep . [ label "value" . store value ] )* . eol ]


(******************************************************************************
 * DEVICES
 ******************************************************************************)

let dev_re = /dev(i(ce?)?)?/i

let dev_containers_re = /containers/i
let dev_partitions_re = /partitions/i

let dev_containers = [ del dev_containers_re "containers" . label "containers" ]
let dev_partitions = [ del dev_partitions_re "partitions" . label "partitions" ]
let dev_device = [ label "name". store ( value - (dev_containers_re | dev_partitions_re)) ]

(* Strictly there must be at least 1 device, but we err on the side of parsing
*)
let dev_devices = ( value_sep . ( dev_containers
                                  | dev_partitions
                                  | dev_device ) )*

let device = [ del dev_re "DEVICE" . label "device" . dev_devices . eol ]


(******************************************************************************
 * ARRAY
 ******************************************************************************)

let array_re  = /arr(ay?)?/i

let arr_auto_re         = /auto/i
let arr_bitmap_re       = /bitmap/i
let arr_container_re    = /container/i
let arr_devices_re      = /devices/i
let arr_disks_re        = /disks/i (* Undocumented *)
let arr_level_re        = /level/i
let arr_member_re       = /member/i
let arr_metadata_re     = /metadata/i
let arr_name_re         = /name/i
let arr_num_devices_re  = /num-devices/i
let arr_spare_group_re  = /spare-group/i
let arr_spares_re       = /spares/i
let arr_super_minor_re  = /super-minor/i
let arr_uuid_re         = /uuid/i

let arr_devicename      = [ store value_no_eq . label "devicename" ]

let arr_auto        = keyvalue arr_auto_re "auto" "AUTO"
let arr_bitmap      = keyvalue arr_bitmap_re "bitmap" "BITMAP"
let arr_container   = keyvalue arr_container_re "container" "CONTAINER"
let arr_devices     = keyvalue arr_devices_re "devices" "DEVICES"
let arr_disks       = keyvalue arr_disks_re "disks" "DISKS"
let arr_level       = keyvalue arr_level_re "level" "LEVEL"
let arr_member      = keyvalue arr_member_re "member" "MEMBER"
let arr_metadata    = keyvalue arr_metadata_re "metadata" "METADATA"
let arr_name        = keyvalue arr_name_re "name" "NAME"
let arr_num_devices = keyvalue arr_num_devices_re "num-devices" "NUM-DEVICES"
let arr_spare_group = keyvalue arr_spare_group_re "spare-group" "SPARE-GROUP"
let arr_spares      = keyvalue arr_spares_re "spares" "SPARES"
let arr_super_minor = keyvalue arr_super_minor_re "super-minor" "SUPER-MINOR"
let arr_uuid        = keyvalue arr_uuid_re "uuid" "UUID"

let arr_options = ( value_sep . ( arr_devicename
                                  | arr_auto
                                  | arr_bitmap
                                  | arr_container
                                  | arr_devices
                                  | arr_disks
                                  | arr_level
                                  | arr_member
                                  | arr_metadata
                                  | arr_name
                                  | arr_num_devices
                                  | arr_spare_group
                                  | arr_spares
                                  | arr_super_minor
                                  | arr_uuid ) )*

let array  = [ del array_re "ARRAY" . label "array" . arr_options . eol ]


(******************************************************************************
 * MAILADDR
 ******************************************************************************)

let mailaddr_re = /mai(l(a(d(dr?)?)?)?)?/i

(* We intentionally allow multiple mailaddr values here, even though this is
invalid and would produce a warning. This is better than not parsing the file.
*)
let mailaddr = simplevalue mailaddr_re "mailaddr" "MAILADDR"


(******************************************************************************
 * MAILFROM
 ******************************************************************************)

(* N.B. MAILFROM can only be abbreviated to 5 characters *)
let mailfrom_re = /mailf(r(om?)?)?/i

let mailfrom = [ del mailfrom_re "MAILFROM" . label "mailfrom"
                 . ( value_sep . [ label "value" . store value ] )* . eol ]


(******************************************************************************
 * PROGRAM
 ******************************************************************************)

let program_re = /pro(g(r(am?)?)?)?/i

let program = simplevalue program_re "program" "PROGRAM"


(******************************************************************************
 * CREATE
 ******************************************************************************)

let create_re = /cre(a(te?)?)?/i

let cre_auto_re     = /auto/i
let cre_owner_re    = /owner/i
let cre_group_re    = /group/i
let cre_mode_re     = /mode/i
let cre_metadata_re = /metadata/i
let cre_symlinks_re = /symlinks/i

let cre_auto        = keyvalue cre_auto_re "auto" "AUTO"
let cre_group       = keyvalue cre_group_re "group" "GROUP"
let cre_metadata    = keyvalue cre_metadata_re "metadata" "METADATA"
let cre_mode        = keyvalue cre_mode_re "mode" "MODE"
let cre_owner       = keyvalue cre_owner_re "owner" "OWNER"
let cre_symlinks    = keyvalue cre_symlinks_re "symlinks" "SYMLINKS"

let cre_options = ( value_sep . ( arr_auto
                                  | cre_owner
                                  | cre_group
                                  | cre_mode
                                  | cre_metadata
                                  | cre_symlinks) )*

let create  = [ del create_re "CREATE" . label "create" . cre_options . eol ]


(******************************************************************************
 * HOMEHOST
 ******************************************************************************)

let homehost_re = /hom(e(h(o(st?)?)?)?)?/i

let homehost = simplevalue homehost_re "homehost" "HOMEHOST"


(******************************************************************************
 * AUTO
 ******************************************************************************)

let auto_re = /auto?/i

let aut_plus        = [ key "+" . store value ]
let aut_minus       = [ key "-" . store value ]
let aut_homehost    = [ del /homehost/i "homehost" . label "homehost" ]

let aut_list = ( value_sep . ( aut_plus | aut_minus | aut_homehost ) )*

let auto = [ del auto_re "AUTO" . label "auto" . aut_list . eol ]


(******************************************************************************
 * POLICY and PART-POLICY
 ******************************************************************************)

(* PART-POLICY is undocumented. A cursory inspection of the parsing code
suggests it's parsed the same way as POLICY, but treated slightly differently
thereafter. *)

let policy_re = /pol(i(cy?)?)?/i
let part_policy_re = /par(t(-(p(o(l(i(cy?)?)?)?)?)?)?)?/i

(* Unlike everything else, policy keys are matched case sensitive. This means we
don't have to mess around with explicit option matching, as the match string is
fixed for a working configuration. *)

let pol_option (act:string) =
    [ del ( act . "=" ) ( act . "=" ) . label act . store value ]

let pol_options = ( value_sep . [ key value_no_eq_sl . del "=" "="
                                  . store value ] )*

let policy      = [ del policy_re "POLICY" . label "policy"
                    . pol_options . eol ]
let part_policy = [ del part_policy_re "PART-POLICY" . label "part-policy"
                    . pol_options . eol ]


(******************************************************************************
 * LENS
 ******************************************************************************)

let lns = (comment
           | empty
           | device
           | array
           | mailaddr
           | mailfrom
           | program
           | create
           | homehost
           | auto
           | policy
           | part_policy )*

let filter     = incl "/etc/mdadm.conf"
               . incl "/etc/mdadm/mdadm.conf"

let xfm        = transform lns filter
