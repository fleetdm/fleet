(* Module: AptCacherNGSecurity

   Lens for config files like the one found in
   /etc/apt-cacher-ng/security.conf


   About: License
   Copyright 2013 Erik B. Andersen; this file is licenced under the LGPL v2+.
*)
module AptCacherNGSecurity =
	autoload xfm

	(* Define a Username/PW pair *)
	let authpair = [ key /[^ \t:\/]*/ . del /:/ ":" . store /[^: \t\n]*/ ]

	(* Define a record. So far as I can tell, the only auth level supported is Admin *)
	let record = [ key "AdminAuth". del /[ \t]*:[ \t]*/ ": ". authpair . Util.del_str "\n"]

	(* Define the basic lens *)
	let lns = ( record | Util.empty | Util.comment )*

	let filter = incl "/etc/apt-cacher-ng/security.conf"
		. Util.stdexcl

	let xfm = transform lns filter
