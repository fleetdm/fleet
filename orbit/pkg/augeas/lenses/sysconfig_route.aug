(*
Module: Sysconfig_Route
  Parses /etc/sysconfig/network-scripts/route-${device}

Author: Stephen P. Schaefer

About: Reference
    This lens allows manipulation of *one* IPv4 variant of the
/etc/sysconfig/network-scripts/route-${device} script found in RHEL5+, CentOS5+
and Fedora.

    The variant handled consists of lines like
    "destination_subnet/cidr via router_ip", e.g.,
    "10.132.11.0/24 via 10.10.2.1"
    
    There are other variants; if you use them, please enhance this lens.
    
    The natural key would be "10.132.11.0/24" with value "10.10.2.1", but since
    augeas cannot deal with slashes in the key value, I reverse them, so that the
    key is "10.10.2.1[1]" (and "10.10.2.1[2]"... if multiple subnets are reachable
    via that router).

About: License
   This file is licenced under the LGPL v2+, like the rest of Augeas.

About: Lens Usage
  Sample usage of this lens in augtool

    * Set the first subnet reachable by a router reachable on the eth0 subnet
      > set /files/etc/sysconfig/network-scripts/route-eth0/10.10.2.1[1] 172.16.254.0/24
    * List all the subnets reachable by a router reachable on eth0 subnet
      > match /files/etc/sysconfig/network-scripts/route-eth0/10.10.2.1

About: Configuration files
  This lens applies to /etc/sysconfig/network-scripts/route-*

About: Examples
  The <Test_Sysconfig_Route> file contains various examples and tests.
*)

module Sysconfig_Route =
	autoload xfm

(******************************************************************************
 * Group:                        USEFUL PRIMITIVES
 ******************************************************************************)

(* Variable: router
   A router *)
let router = Rx.ipv4
(* Variable: cidr
   A subnet mask can be 0 to 32 bits *)
let cidr = /(3[012]|[12][0-9]|[0-9])/
(* Variable: subnet
   Subnet specification *)
let subnet = Rx.ipv4 . "/" . cidr

(******************************************************************************
 * Group:                        ENTRY TYPES
 ******************************************************************************)

(* View: entry
   One route *)
let entry = [ store subnet . del /[ \t]*via[ \t]*/ " via "
            . key router . Util.del_str "\n" ]

(******************************************************************************
 * Group:                        LENS AND FILTER
 ******************************************************************************)

(* View: lns *)
let lns = entry+

(* View: filter *)
let filter = incl "/etc/sysconfig/network-scripts/route-*"
	   . Util.stdexcl

let xfm = transform lns filter
