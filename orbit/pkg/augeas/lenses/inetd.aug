(* inetd.conf lens definition for Augeas
   Auther: Matt Palmer <mpalmer@hezmatt.org>

   Copyright (C) 2009 Matt Palmer, All Rights Reserved

   This program is free software: you can redistribute it and/or modify it
   under the terms of the GNU Lesser General Public License version 2.1 as
   published by the Free Software Foundation.

   This program is distributed in the hope that it will be useful, but
   WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU General
   Public License for more details.

   You should have received a copy of the GNU General Public License along
   with this program.  If not, see <http://www.gnu.org/licenses/>.

This lens parses /etc/inetd.conf.  The current format is based on the
syntax documented in the inetd manpage shipped with Debian's openbsd-inetd
package version 0.20080125-2.  Apologies if your inetd.conf doesn't follow
the same format.

Each top-level entry will have a key being that of the service name (the
first column in the service definition, which is the name or number of the
port that the service should listen on).  The attributes for the service all
sit under that.  In regular Augeas style, the order of the attributes
matter, and attempts to set things in a different order will fail miserably.
The defined attribute names (and the order in which they must appear) are as
follows (with mandatory parameters indicated by [*]):

address -- a sequence of IP addresses or hostnames on which this service
	should listen.

socket[*] -- The type of the socket that will be created (either stream or
	dgram, although the lens doesn't constrain the possibilities here)

protocol[*] -- The socket protocol.  I believe that the usual possibilities
	are "tcp", "udp", or "unix", but no restriction is made on what you
	can actually put here.

sndbuf -- Specify a non-default size for the send buffer of the connection.

rcvbuf -- Specify a non-default size for the receive buffer of the connection.

wait[*] -- Whether to wait for new connections ("wait"), or just terminate
	immediately ("nowait").

max -- The maximum number of times that a service can be invoked in one minute.

user[*] -- The user to run the service as.

group -- A group to set the running service to, rather than the primary
	group of the previously specified user.

command[*] -- What program to run.

arguments -- A sequence of arguments to pass to the command.

In addition to this straightforward tree, inetd has the ability to set
"default" listen addresses; this is a little used feature which nonetheless
comes in handy sometimes.  The key for entries of this type is "address"
, and the subtree should be a sequence of addresses.  "*" can
always be used to return the default behaviour of listening on INADDR_ANY.

*)

module Inetd =
	autoload xfm

	(***************************
	 * PRIMITIVES
	 ***************************)

	(* Store whitespace *)
	let wsp = del /[ \t]+/ " "
	let sep = del /[ \t]+/ "	"
	let owsp(t:string) = del /[ \t]*/ t

	(* It's the end of the line as we know it... doo, doo, dooooo *)
	let eol = Util.eol

	(* In the beginning, the earth was without form, and void *)
	let empty = Util.empty

	let comment = Util.comment

	let del_str = Util.del_str

	let address = [ seq "addrseq" . store /([a-zA-Z0-9.-]+|\[[A-Za-z0-9:?*%]+\]|\*)/ ]
	let address_list = ( counter "addrseq" . (address . del_str ",")* . address )

	let argument = [ seq "argseq" . store /[^ \t\n]+/ ]
	let argument_list = ( counter "argseq" . [ label "arguments" . (argument . wsp)* . argument ] )

	(***************************
	 * ELEMENTS
	 ***************************)

	let service (l:string) = ( label l . [label "address" . address_list . del_str ":" ]? . store /[^ \t\n\/:#]+/ )

	let socket = [ label "socket" . store /[^ \t\n#]+/ ]

	let protocol = ( [ label "protocol" . store /[^ \t\n,#]+/ ]
	                 . [ del_str "," . key /sndbuf/ . del_str "=" . store /[^ \t\n,]+/ ]?
	                 . [ del_str "," . key /rcvbuf/ . del_str "=" . store /[^ \t\n,]+/ ]?
	               )

	let wait = ( [ label "wait" . store /(wait|nowait)/ ]
	             . [ del_str "." . label "max" . store /[0-9]+/ ]?
	           )

	let usergroup = ( [ label "user" . store /[^ \t\n:.]+/ ]
	                  . [ del /[:.]/ ":" . label "group" . store /[^ \t\n:.]+/ ]?
	                )

	let command = ( [ label "command" . store /[^ \t\n]+/ ]
	                . (wsp . argument_list)?
	              )

	(***************************
	 * SERVICE LINES
	 ***************************)

	let service_line = [ service "service"
	                     . sep
	                     . socket
	                     . sep
	                     . protocol
	                     . sep
	                     . wait
	                     . sep
	                     . usergroup
	                     . sep
	                     . command
	                     . eol
	                   ]


	(***************************
	 * RPC LINES
	 ***************************)

	let rpc_service = service "rpc_service" . Util.del_str "/"
                        . [ label "version" . store Rx.integer ]

        let rpc_endpoint = [ label "endpoint-type" . store Rx.word ]
        let rpc_protocol = Util.del_str "rpc/"
                         . (Build.opt_list
                             [label "protocol" . store /[^ \t\n,#]+/ ]
                             Sep.comma)

	let rpc_line = [ rpc_service
	                     . sep
	                     . rpc_endpoint
	                     . sep
	                     . rpc_protocol
	                     . sep
	                     . wait
	                     . sep
	                     . usergroup
	                     . sep
	                     . command
	                     . eol
	                   ]


	(***************************
	 * DEFAULT LISTEN ADDRESSES
	 ***************************)

	let default_listen_address = [ label "address"
	                               . address_list
	                               . del_str ":"
	                               . eol
	                             ]

	(***********************
	 * LENS / FILTER
	 ***********************)

	let lns = (comment|empty|service_line|rpc_line|default_listen_address)*

	let filter = incl "/etc/inetd.conf"

	let xfm = transform lns filter
