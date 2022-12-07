(*
Module: Syslog
  parses /etc/syslog.conf

Author: Mathieu Arnold <mat@FreeBSD.org>

About: Reference
  This lens tries to keep as close as possible to `man 5 resolv.conf` where possible.
  An online source being :
  http://www.freebsd.org/cgi/man.cgi?query=syslog.conf&sektion=5

About: Licence
  This file is licensed under the BSD License.

About: Lens Usage
   To be documented

About: Configuration files
  This lens applies to /etc/syslog.conf. See <filter>.

 *)
module Syslog =
  autoload xfm

	(************************************************************************
	 * Group:                 USEFUL PRIMITIVES
	 *************************************************************************)

	(* Group: Comments and empty lines *)

	(* Variable: empty *)
        let empty      = Util.empty
	(* Variable: eol *)
        let eol        = Util.eol
	(* Variable: sep_tab *)
        let sep_tab    = del /([ \t]+|[ \t]*\\\\\n[ \t]*)/ "\t"

	(* Variable: sep_tab_opt *)
        let sep_tab_opt = del /([ \t]*|[ \t]*\\\\\n[ \t]*)/ ""

	(* View: comment
	  Map comments into "#comment" nodes
	  Can't use Util.comment as #+ and #! have a special meaning.
      However, '# !' and '# +' have no special meaning so they should be allowed.
     *)

	let comment_gen (space:regexp) (sto:regexp) =
      [ label "#comment" . del (Rx.opt_space . "#" . space) "# "
        . store sto . eol ]

	let comment =
		let comment_withsign = comment_gen Rx.space /([!+-].*[^ \t\n]|[!+-])/
	 in let comment_nosign = comment_gen Rx.opt_space /([^ \t\n+!-].*[^ \t\n]|[^ \t\n+!-])/
	 in comment_withsign | comment_nosign

	(* Group: single characters macro *)

        (* Variable: comma
	 Deletes a comma and default to it
	 *)
	let comma      = sep_tab_opt . Util.del_str "," . sep_tab_opt
	(* Variable: colon
	 Deletes a colon and default to it
	 *)
	let colon      = sep_tab_opt . Util.del_str ":" . sep_tab_opt
	(* Variable: semicolon
	 Deletes a semicolon and default to it
	 *)
	let semicolon  = sep_tab_opt . Util.del_str ";" . sep_tab_opt
	(* Variable: dot
	 Deletes a dot and default to it
	 *)
	let dot        = Util.del_str "."
	(* Variable: pipe
	 Deletes a pipe and default to it
	 *)
	let pipe       = Util.del_str "|"
	(* Variable: plus
	 Deletes a plus and default to it
	 *)
	let plus       = Util.del_str "+"
	(* Variable: bang
	 Deletes a bang and default to it
	 *)
	let bang       = Util.del_str "!"

	(* Variable: opt_hash
	  deletes an optional # sign
	  *)
	let opt_hash   = del /#?/ ""
	(* Variable: opt_plus
	  deletes an optional + sign
	  *)
	let opt_plus   = del /\+?/ ""

	(* Group: various macros *)

	(* Variable: word
	  our version can't start with [_.-] because it would mess up the grammar
	  *)
	let word      = /[A-Za-z0-9][A-Za-z0-9_.-]*/

	(* Variable: comparison
	  a comparison is an optional ! with optionally some of [<=>]
	  *)
        let comparison = /(!|[<=>]+|![<=>]+)/

	(* Variable: protocol
	  @ means UDP
    @@ means TCP
	  *)
        let protocol      = /@{1,2}/

	(* Variable: token
	  alphanum or "*"
	  *)
        let token      = /([A-Za-z0-9]+|\*)/

	(* Variable: file_r
	 a file begins with a / and get almost anything else after
	 *)
	let file_r     = /\/[^ \t\n;]+/

	(* Variable: loghost_r
	 Matches a hostname, that is labels speparated by dots, labels can't
	 start or end with a "-".  maybe a bit too complicated for what it's worth *)
	let loghost_r = /[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?)*/ |
                    "[" . Rx.ipv6 . "]"

	(* Group: Function *)

	(* View: label_opt_list
	 Uses Build.opt_list to generate a list of labels

	 Parameters:
	  l:string - the label name
	  r:lens   - the lens going after the label
	  s:lens   - the separator lens passed to Build.opt_list
	 *)
	let label_opt_list (l:string) (r:lens) (s:lens) = Build.opt_list [ label l . r ] s

	(* View: label_opt_list_or
	 Either label_opt_list matches something or it emits a single label
	 with the "or" string.

	 Parameters:
	  l:string  - the label name
	  r:lens    - the lens going after the label
	  s:lens    - the separator lens passed to Build.opt_list
	  or:string - the string used if the label_opt_list does not match anything
	 *)
	let label_opt_list_or (l:string) (r:lens) (s:lens) (or:string) =
	  ( label_opt_list l r s | [ label l . store or ] )


	(************************************************************************
	 * Group:                 LENSE DEFINITION
	 *************************************************************************)

	(* Group: selector *)

	(* View: facilities
	 a list of facilities, separated by commas
	 *)
	let facilities = label_opt_list "facility" (store token) comma

	(* View: selector
	 a selector is a list of facilities, an optional comparison and a level
	 *)
	let selector = facilities . dot .
                       [ label "comparison" . store comparison]? .
                       [ label "level" . store token ]

	(* View: selectors
	 a list of selectors, separated by semicolons
	 *)
        let selectors = label_opt_list "selector" selector semicolon

	(* Group: action *)

	(* View: file
	 a file may start with a "-" meaning it does not gets sync'ed everytime
	 *)
	let file = [ Build.xchgs "-" "no_sync" ]? . [ label "file" . store file_r ]

	(* View: loghost
	 a loghost is an @  sign followed by the hostname and a possible port
	 *)
	let loghost = [label "protocol" . store protocol] . [ label "hostname" . store loghost_r ] .
	    (colon . [ label "port" . store /[0-9]+/ ] )?

	(* View: users
	 a list of users or a "*"
	 *)
	let users = label_opt_list_or "user" (store word) comma "*"

	(* View: logprogram
	 a log program begins with a pipe
	 *)
	let logprogram = pipe . [ label "program" . store /[^ \t\n][^\n]+[^ \t\n]/ ]

	(* View: discard
	 discards matching messages
	 *)
	let discard = [ label "discard" . Util.del_str "~" ]

	(* View: action
	 an action is either a file, a host, users, a program, or discard
	 *)
        let action = (file | loghost | users | logprogram | discard)

	(* Group: Entry *)

	(* View: entry
	 an entry contains selectors and an action
	 *)
        let entry = [ label "entry" .
	    selectors . sep_tab .
	    [ label "action" . action ] . eol ]

	(* View: entries
	 entries are either comments/empty lines or entries
	 *)
	let entries = (empty | comment | entry )*

	(* Group: Program matching *)

	(* View: programs
	 a list of programs
	 *)
	let programs = label_opt_list_or "program" (store word) comma "*"

	(* View: program
	 a program begins with an optional hash, a bang, and an optional + or -
	 *)
	let program = [ label "program" . opt_hash . bang .
	      ( opt_plus | [ Build.xchgs "-" "reverse" ] ) .
	      programs . eol .  entries ]

	(* Group: Hostname maching *)

	(* View: hostnames
	 a list of hostnames
	 *)
	let hostnames = label_opt_list_or "hostname" (store Rx.word) comma "*"

	(* View: hostname
	 a program begins with an optional hash, and a + or -
	 *)
	let hostname = [ label "hostname" . opt_hash .
	      ( plus | [ Build.xchgs "-" "reverse" ] ) .
	      hostnames . eol .  entries ]

	(* Group: Top of the tree *)

    let include =
      [ key "include" . sep_tab . store file_r . eol ]

	(* View: lns
	 generic entries then programs or hostnames matching blocs
	 *)
        let lns = entries . ( program | hostname | include )*

	(* Variable: filter
	 all you need is /etc/syslog.conf
	 *)
        let filter = incl "/etc/syslog.conf"

        let xfm = transform lns filter
