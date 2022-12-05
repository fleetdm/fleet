(*
Module: Sshd
  Parses /etc/ssh/sshd_config

Author: David Lutterkort lutter@redhat.com
        Dominique Dumont dominique.dumont@hp.com

About: Reference
  sshd_config man page.
  See http://www.openbsd.org/cgi-bin/man.cgi?query=sshd_config&sektion=5

About: License
  This file is licensed under the LGPL v2+.

About: Lens Usage
  Sample usage of this lens in augtool:

    * Get your current setup
      > print /files/etc/ssh/sshd_config
      ...

    * Set X11Forwarding to "no"
      > set /files/etc/ssh/sshd_config/X11Forwarding "no"

  More advanced usage:

    * Set a Match section
      > set /files/etc/ssh/sshd_config/Match[1]/Condition/User "foo"
      > set /files/etc/ssh/sshd_config/Match[1]/Settings/X11Forwarding "yes"

  Saving your file:

      > save


About: CAVEATS

  In sshd_config, Match blocks must be located at the end of the file.
  This means that any new "global" parameters (i.e. outside of a Match
  block) must be written before the first Match block. By default,
  Augeas will write new parameters at the end of the file.

  I.e. if you have a Match section and no ChrootDirectory parameter,
  this command:

     > set /files/etc/ssh/sshd_config/ChrootDirectory "foo"

  will be stored in a new node after the Match section and Augeas will
  refuse to save sshd_config file.

  To create a new parameter as the right place, you must first create
  a new Augeas node before the Match section:

     > ins ChrootDirectory before /files/etc/ssh/sshd_config/Match

  Then, you can set the parameter

     > set /files/etc/ssh/sshd_config/ChrootDirectory "foo"


About: Configuration files
  This lens applies to /etc/ssh/sshd_config

*)

module Sshd =
   autoload xfm

   let eol = del /[ \t]*\n/ "\n"

   let sep = del /[ \t=]+/ " "

   let indent = del /[ \t]*/ "  "

   let key_re = /[A-Za-z0-9]+/
         - /MACs|Match|AcceptEnv|Subsystem|Ciphers|((GSSAPI|)Kex|HostKey|CASignature)Algorithms|PubkeyAcceptedKeyTypes|(Allow|Deny)(Groups|Users)/i

   let comment = Util.comment
   let comment_noindent = Util.comment_noindent
   let empty = Util.empty

   let array_entry (kw:regexp) (sq:string) =
     let bare = Quote.do_quote_opt_nil (store /[^"' \t\n=]+/) in
     let quoted = Quote.do_quote (store /[^"'\n]*[ \t]+[^"'\n]*/) in
     [ key kw
       . ( [ sep . seq sq . bare ] | [ sep . seq sq . quoted ] )*
       . eol ]

   let other_entry =
     let value = store /[^ \t\n=]+([ \t=]+[^ \t\n=]+)*/ in
     [ key key_re . sep . value . eol ]

   let accept_env = array_entry /AcceptEnv/i "AcceptEnv"

   let allow_groups = array_entry /AllowGroups/i "AllowGroups"
   let allow_users = array_entry /AllowUsers/i "AllowUsers"
   let deny_groups = array_entry /DenyGroups/i "DenyGroups"
   let deny_users = array_entry /DenyUsers/i "DenyUsers"

   let subsystemvalue =
     let value = store (/[^ \t\n=](.*[^ \t\n=])?/) in
     [ key /[A-Za-z0-9\-]+/ . sep . value . eol ]

   let subsystem =
     [ key /Subsystem/i .  sep .  subsystemvalue ]

   let list (kw:regexp) (sq:string) =
     let value = store /[^, \t\n=]+/ in
     [ key kw . sep .
         [ seq sq . value ] .
         ([ seq sq . Util.del_str "," . value])* .
         eol ]

   let macs = list /MACs/i "MACs"

   let ciphers = list /Ciphers/i "Ciphers"

   let kexalgorithms = list /KexAlgorithms/i "KexAlgorithms"

   let hostkeyalgorithms = list /HostKeyAlgorithms/i "HostKeyAlgorithms"

   let gssapikexalgorithms = list /GSSAPIKexAlgorithms/i "GSSAPIKexAlgorithms"

   let casignaturealgorithms = list /CASignatureAlgorithms/i "CASignatureAlgorithms"

   let pubkeyacceptedkeytypes = list /PubkeyAcceptedKeyTypes/i "PubkeyAcceptedKeyTypes"

   let entry = accept_env | allow_groups | allow_users
             | deny_groups | subsystem | deny_users
             | macs | ciphers | kexalgorithms | hostkeyalgorithms
             | gssapikexalgorithms | casignaturealgorithms
             | pubkeyacceptedkeytypes | other_entry

   let condition_entry =
    let k = /[A-Za-z0-9]+/ in
    let no_spc = Quote.do_dquote_opt (store  /[^"' \t\n=]+/) in
    let spc = Quote.do_quote (store /[^"'\t\n]* [^"'\t\n]*/) in
      [ sep . key k . sep . no_spc ]
    | [ sep . key k . sep . spc ]

   let match_cond =
     [ label "Condition" . condition_entry+ . eol ]

   let match_entry = indent . (entry | comment_noindent)
                   | empty

   let match =
     [ key /Match/i . match_cond
        . [ label "Settings" .  match_entry+ ]
     ]

  let lns = (entry | comment | empty)* . match*

  let filter = (incl "/etc/ssh/sshd_config" )
               . ( incl "/etc/ssh/sshd_config.d/*.conf" )

  let xfm = transform lns filter

(* Local Variables: *)
(* mode: caml       *)
(* End:             *)
