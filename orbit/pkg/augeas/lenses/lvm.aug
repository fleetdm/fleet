(*
Module: LVM
  Parses LVM metadata.

Author: Gabriel de Perthuis	      <g2p.code+augeas@gmail.com>

About: License
  This file is licensed under the LGPL v2+.

About: Configuration files
  This lens applies to files in /etc/lvm/backup and /etc/lvm/archive.

About: Examples
  The <Test_LVM> file contains various examples and tests.
*)

module LVM =
	autoload xfm

	(* See lvm2/libdm/libdm-config.c for tokenisation;
	 * libdm uses a blacklist but I prefer the safer whitelist approach. *)
	(* View: identifier
	 * The left hand side of a definition *)
	let identifier = /[a-zA-Z0-9_-]+/

	(* strings can contain backslash-escaped dquotes, but I don't know
	 * how to get the message across to augeas *)
	let str = [label "str". Quote.do_dquote (store /([^\"]|\\\\.)*/)]
	let int = [label "int". store Rx.relinteger]
	(* View: flat_literal
	 * A literal without structure *)
	let flat_literal = int|str

	(* allow multiline and mixed int/str, used for raids and stripes *)
	(* View: list
	 * A list containing flat literals *)
	let list = [
		  label "list" . counter "list"
		. del /\[[ \t\n]*/ "["
		.([seq "list". flat_literal . del /,[ \t\n]*/ ", "]*
				. [seq "list". flat_literal . del /[ \t\n]*/ ""])?
		. Util.del_str "]"]

	(* View: val
	 * Any value that appears on the right hand side of an assignment *)
	let val = flat_literal | list

	(* View: nondef
	 * A line that doesn't contain a statement *)
	let nondef =
		  Util.empty
		| Util.comment

	(* Build.block couldn't be reused, because of recursion and
	 * a different philosophy of whitespace handling. *)
	(* View: def
	 * An assignment, or a block containing definitions *)
	let rec def = [
		  Util.indent . key identifier . (
			   del /[ \t]*\{\n/ " {\n"
			  .[label "dict".(nondef | def)*]
			  . Util.indent . Util.del_str "}\n"
			  |Sep.space_equal . val . Util.comment_or_eol)]

	(* View: lns
	 * The main lens *)
	let lns = (nondef | def)*

	let filter =
		  incl "/etc/lvm/archive/*.vg"
		. incl "/etc/lvm/backup/*"
		. incl "/etc/lvm/lvm.conf"
		. Util.stdexcl

	let xfm = transform lns filter
