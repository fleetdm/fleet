(*
Module: Ssh
  Parses ssh client configuration

Author: Jiri Suchomel <jsuchome@suse.cz>

About: Reference
    ssh_config man page

About: License
    This file is licensed under the GPL.

About: Lens Usage
  Sample usage of this lens in augtool

(start code)
augtool> set /files/etc/ssh/ssh_config/Host example.com
augtool> set /files/etc/ssh/ssh_config/Host[.='example.com']/RemoteForward/machine1:1234 machine2:5678
augtool> set /files/etc/ssh/ssh_config/Host[.='example.com']/Ciphers/1 aes128-ctr
augtool> set /files/etc/ssh/ssh_config/Host[.='example.com']/Ciphers/2 aes192-ctr
(end code)

*)

module Ssh =
    autoload xfm

(************************************************************************
 * Group:                 USEFUL PRIMITIVES
 *************************************************************************)

    let eol = Util.doseol
    let spc = Util.del_ws_spc
    let spc_eq = del /[ \t]+|[ \t]*=[ \t]*/ " "
    let comment = Util.comment
    let empty = Util.empty
    let comma = Util.del_str ","
    let indent = Util.indent
    let value_to_eol = store Rx.space_in
    let value_to_spc = store /[^ \t\r\n=][^ \t\r\n]*/
    let value_to_comma = store /[^, \t\r\n=][^, \t\r\n]*/


(************************************************************************
 * Group:                 ENTRIES
 *************************************************************************)

    let array_entry (k:regexp) =
        [ indent . key k . counter "array_entry"
         . [ spc . seq "array_entry" . value_to_spc]* . eol ]

    let commas_entry (k:regexp) =
         let value = [ seq "commas_entry" . value_to_comma]
      in [ indent . key k . counter "commas_entry" . spc_eq .
           Build.opt_list value comma . eol ]

    let spaces_entry (k:regexp) =
         let value = [ seq "spaces_entry" . value_to_spc ]
      in [ indent . key k . counter "spaces_entry" . spc_eq .
           Build.opt_list value spc . eol ]

    let fw_entry (k:regexp) = [ indent . key k . spc_eq .
	    [ key /[^ \t\r\n\/=][^ \t\r\n\/]*/ . spc . value_to_eol . eol ]]

    let send_env = array_entry /SendEnv/i

    let proxy_command = [ indent . key /ProxyCommand/i . spc . value_to_eol . eol ]

    let remote_fw = fw_entry /RemoteForward/i
    let local_fw = fw_entry /LocalForward/i

    let ciphers = commas_entry /Ciphers/i
    let macs	= commas_entry /MACs/i
    let algorithms = commas_entry /(HostKey|Kex)Algorithms/i
    let pubkey_accepted_key_types = commas_entry /PubkeyAcceptedKeyTypes/i

    let global_knownhosts_file = spaces_entry /GlobalKnownHostsFile/i

    let rekey_limit = [ indent . key /RekeyLimit/i . spc_eq .
                        [ label "amount" . value_to_spc ] .
                        [ spc . label "duration" . value_to_spc ]? . eol ]

    let special_entry = send_env
	                    | proxy_command
	                    | remote_fw
	                    | local_fw
	                    | macs
	                    | ciphers
	                    | algorithms
	                    | pubkey_accepted_key_types
                        | global_knownhosts_file
                        | rekey_limit

    let key_re = /[A-Za-z0-9]+/
               - /SendEnv|Host|Match|ProxyCommand|RemoteForward|LocalForward|MACs|Ciphers|(HostKey|Kex)Algorithms|PubkeyAcceptedKeyTypes|GlobalKnownHostsFile|RekeyLimit/i


    let other_entry = [ indent . key key_re
                    . spc_eq . value_to_spc . eol ]

    let entry = comment | empty
              | special_entry
	            | other_entry

    let host = [ key /Host/i . spc . value_to_eol . eol . entry* ]


   let condition_entry =
    let k = /[A-Za-z0-9]+/ in
    let no_spc = Quote.do_dquote_opt (store  /[^"' \t\r\n=]+/) in
    let with_spc = Quote.do_quote (store /[^"'\t\r\n]* [^"'\t\r\n]*/) in
      [ spc . key k . spc . no_spc ]
    | [ spc . key k . spc . with_spc ]

   let match_cond =
     [ label "Condition" . condition_entry+ . eol ]

   let match_entry = entry

   let match =
     [ key /Match/i . match_cond
        . [ label "Settings" .  match_entry+ ]
     ]


(************************************************************************
 * Group:                 LENS
 *************************************************************************)

    let lns = entry* . (host | match)*

    let xfm = transform lns (incl "/etc/ssh/ssh_config" .
                             incl (Sys.getenv("HOME") . "/.ssh/config") .
                             incl "/etc/ssh/ssh_config.d/*.conf")
