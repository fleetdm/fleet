(* Python paste config file lens for Augeas
   Author: Dan Prince <dprince@redhat.com>
*)
module PythonPaste =
  autoload xfm

(************************************************************************
 * INI File settings
 *************************************************************************)

let comment  = IniFile.comment "#" "#"

let sep        = IniFile.sep "=" "="

let eol     = Util.eol

(************************************************************************
 *                        ENTRY
 *************************************************************************)

let url_entry    = /\/[\/A-Za-z0-9.-_]* ?[:|=] [A-Za-z0-9.-_]+/

let set_kw = [ Util.del_str "set" . Util.del_ws_spc . label "@set" ]

let no_inline_comment_entry (kw:regexp) (sep:lens) (comment:lens)
                       = [ set_kw? . key kw . sep . IniFile.sto_to_eol? . eol ]
                         | comment
                         | [ seq "urls" . store url_entry . eol ]

let entry_re           = ( /[A-Za-z][:#A-Za-z0-9._-]+/ )

let entry   = no_inline_comment_entry entry_re sep comment

(************************************************************************
 *                        RECORD
 *************************************************************************)

let title   = IniFile.title IniFile.record_re

let record  = IniFile.record title entry

(************************************************************************
 *                        LENS & FILTER
 *************************************************************************)
let lns     = IniFile.lns record comment

let filter = ((incl "/etc/glance/*.ini")
             . (incl "/etc/keystone/keystone.conf")
             . (incl "/etc/nova/api-paste.ini")
             . (incl "/etc/swift/swift.conf")
             . (incl "/etc/swift/proxy-server.conf")
             . (incl "/etc/swift/account-server/*.conf")
             . (incl "/etc/swift/container-server/*.conf")
             . (incl "/etc/swift/object-server/*.conf"))

let xfm = transform lns filter
