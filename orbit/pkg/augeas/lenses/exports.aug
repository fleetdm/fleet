(* Lens for Linux syntax of NFS exports(5) *)

(*
Module: Exports
    Parses /etc/exports

Author: David Lutterkort <lutter@redhat.com>

About: Description
    /etc/exports contains lines associating a directory with one or
    more hosts, and NFS options for each host.

About: Usage Example

(start code)

    $ augtool
    augtool> ls /files/etc/exports/
    comment[1] = /etc/exports: the access control list for filesystems which may be exported
    comment[2] = to NFS clients.  See exports(5).
    comment[3] = sample /etc/exports file
    dir[1]/ = /
    dir[2]/ = /projects
    dir[3]/ = /usr
    dir[4]/ = /home/joe


    augtool> ls /files/etc/exports/dir[1]
    client[1]/ = master
    client[2]/ = trusty
(end code)

The corresponding line in the file is:

(start code)
	/               master(rw) trusty(rw,no_root_squash)
(end code)

    Digging further:

(start code)
    augtool> ls /files/etc/exports/dir[1]/client[1]
    option = rw

    To add a new entry, you'd do something like this:
(end code)

(start code)
    augtool> set /files/etc/exports/dir[10000] /foo
    augtool> set /files/etc/exports/dir[last()]/client[1] weeble
    augtool> set /files/etc/exports/dir[last()]/client[1]/option[1] ro
    augtool> set /files/etc/exports/dir[last()]/client[1]/option[2] all_squash
    augtool> save
    Saved 1 file(s)
(end code)

    Which creates the line:

(start code)
    /foo weeble(ro,all_squash)
(end code)

About: Limitations
    This lens cannot handle options without a host, as with the last
    example line in "man 5 exports":

	/pub            (ro,insecure,all_squash)

    In this case, though, you can just do:

	/pub            *(ro,insecure,all_squash)

    It also can't handle whitespace before the directory name.
*)

module Exports =
  autoload xfm

  let client_re = /[][a-zA-Z0-9.@*?\/:-]+/

  let eol = Util.eol
  let lbracket  = Util.del_str "("
  let rbracket  = Util.del_str ")"
  let sep_com   = Sep.comma
  let sep_spc   = Sep.space

  let option = [ label "option" . store /[^,)]*/ ]

  let client    = [ label "client" . store client_re .
                    ( Build.brackets lbracket rbracket
                         ( Build.opt_list option sep_com ) )? ]

  let entry = [ label "dir" . store /[^ \t\n#]*/
                . sep_spc . Build.opt_list client sep_spc . eol ]

  let lns = (Util.empty | Util.comment | entry)*

  let xfm = transform lns (incl "/etc/exports")
