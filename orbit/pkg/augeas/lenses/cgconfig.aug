(*
Module: cgconfig
    Parses /etc/cgconfig.conf

Author:
    Ivana Hutarova Varekova <varekova@redhat.com>
    Raphael Pinson          <raphink@gmail.com>

About: Licence
    This file is licensed under the LGPL v2+, like the rest of Augeas.

About: Lens Usage
    Sample usage of this lens in augtool
        * print all mounted cgroups
           print /files/etc/cgconfig.conf/mount

About: Configuration files
    This lens applies to /etc/cgconfig.conf. See <filter>.
 *)

module Cgconfig =
   autoload xfm

   let indent  = Util.indent
   let eol     = Util.eol
   let comment = Util.comment
   let empty   = Util.empty

   let id        = /[a-zA-Z0-9_\/.-]+/
   let name      = /[^#= \n\t{}\/]+/
   let cont_name = /(cpuacct|cpu|devices|ns|cpuset|memory|freezer|net_cls|blkio|hugetlb|perf_event)/
   let role_name = /(admin|task)/
   let id_name   = /(uid|gid|fperm|dperm)/
   let address   = /[^#; \n\t{}]+/
   let qaddress  = address|/"[^#;"\n\t{}]+"/

   let lbracket = del /[ \t\n]*\{/ " {"
   let rbracket = del /[ \t]*\}/ "}"
   let eq       = indent . Util.del_str "=" . indent

(******************************************
 * Function to deal with abc=def; entries
 ******************************************)

   let key_value (key_rx:regexp) (val_rx:regexp) =
     [ indent . key key_rx . eq . store val_rx
         . indent . Util.del_str ";" ]

   (* Function to deal with bracketted entries *)
   let brack_entry_base (lnsa:lens) (lnsb:lens) =
     [ indent . lnsa . lbracket . lnsb . rbracket ]

   let brack_entry_key (kw:regexp) (lns:lens) =
     let lnsa = key kw in
     brack_entry_base lnsa lns

   let brack_entry (kw:regexp) (lns:lens) =
     let full_lns = (lns | comment | empty)* in
     brack_entry_key kw full_lns

(******************************************
 * control groups
 ******************************************)

   let permission_setting = key_value id_name address

(* task setting *)
   let t_info =  brack_entry "task" permission_setting

(* admin setting *)
   let a_info =  brack_entry "admin" permission_setting

(* permissions setting *)
   let perm_info =
     let ce = (comment|empty)* in
     let perm_info_lns = ce .
       ((t_info . ce . (a_info . ce)?)
       |(a_info . ce . (t_info . ce)?))? in
     brack_entry_key "perm" perm_info_lns

   let variable_setting = key_value name qaddress

(* controllers setting *)
   let controller_info =
     let lnsa = label "controller" . store cont_name in
     let lnsb = ( variable_setting | comment | empty ) * in
     brack_entry_base lnsa lnsb

(* group { ... } *)
   let group_data  =
     let lnsa = key "group" . Util.del_ws_spc . store id in
     let lnsb = ( perm_info | controller_info | comment | empty )* in
     brack_entry_base lnsa lnsb


(*************************************************
 * mount point
 *************************************************)

(* controller = mount_point; *)
   let mount_point = key_value name address

(* mount { .... } *)
   let mount_data = brack_entry "mount" mount_point


(****************************************************
 * namespace
 ****************************************************)

(* controller = cgroup; *)
   let namespace_instance = key_value name address


(* namespace { .... } *)
   let namespace = brack_entry "namespace" namespace_instance

   let lns =  ( comment | empty | mount_data | group_data | namespace )*

   let xfm = transform lns (incl "/etc/cgconfig.conf")
