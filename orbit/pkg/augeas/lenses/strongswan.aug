(*
Module: Strongswan
  Lens for parsing strongSwan configuration files

Authors:
  Kaarle Ritvanen <kaarle.ritvanen@datakunkku.fi>

About: Reference
  strongswan.conf(5), swanctl.conf(5)

About: License
  This file is licensed under the LGPL v2+
*)

module Strongswan =

autoload xfm

let ws = del /[\n\t ]*(#[\t ]*\n[\n\t ]*)*/

let rec conf =
	   let keys = /[^\/.\{\}#\n\t ]+/ - /include/
	in let lists = /(crl|oscp)_uris|(local|remote)_(addrs|ts)|vips|pools|(ca)?certs|pubkeys|groups|cert_policy|dns|nbns|dhcp|netmask|server|subnet|split_(in|ex)clude|interfaces_(ignore|use)|preferred/
	in let proposals = /((ah|esp)_)?proposals/
	in let name (pat:lens) (sep:string) =
		pat . Util.del_ws_spc . Util.del_str sep
	in let val = store /[^\n\t ].*/ . Util.del_str "\n" . ws ""
	in let sval = Util.del_ws_spc . val
	in let ival (pat:lens) (end:string) =
		Util.del_opt_ws " " . seq "item" . pat . Util.del_str end
	in let list (l:string) (k:regexp) (v:lens) =
		[ label l . name (store k) "=" . counter "item" .
		  [ ival v "," ]* . [ ival v "\n" ] . ws "" ]
	in let alg = seq "alg" . store /[a-z0-9]+/
in (
	[ Util.del_str "#" . label "#comment" . Util.del_opt_ws " " . val ] |
	[ key "include" . sval ] |
	[ name (key (keys - lists - proposals)) "=" . sval ] |
	list "#list" lists (store /[^\n\t ,][^\n,]*/) |
	list "#proposals" proposals (counter "alg" . [ alg ] . [ Util.del_str "-" . alg ]*) |
	[ name (key keys) "{" . ws "\n" . conf . Util.del_str "}" . ws "\n" ]
)*

let lns = ws "" . conf

let xfm = transform lns (
	incl "/etc/strongswan.d/*.conf" .
	incl "/etc/strongswan.d/**/*.conf" .
	incl "/etc/swanctl/conf.d/*.conf" .
	incl "/etc/swanctl/swanctl.conf"
)
