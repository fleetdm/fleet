(* Dput module for Augeas
   Author: Raphael Pinson <raphink@gmail.com>


   Reference: dput uses Python's ConfigParser:
     http://docs.python.org/lib/module-ConfigParser.html
*)


module Dput =
  autoload xfm


(************************************************************************
 * INI File settings
 *************************************************************************)
let comment  = IniFile.comment IniFile.comment_re IniFile.comment_default

let sep      = IniFile.sep IniFile.sep_re IniFile.sep_default


let setting = "allow_dcut"
            | "allow_non-us_software"
            | "allow_unsigned_uploads"
            | "check_version"
            | "default_host_main"
            | "default_host_non-us"
            | "fqdn"
            | "hash"
            | "incoming"
            | "login"
            | "method"
            | "passive_ftp"
            | "post_upload_command"
            | "pre_upload_command"
            | "progress_indicator"
            | "run_dinstall"
            | "run_lintian"
            | "scp_compress"
            | "ssh_config_options"
            | "allowed_distributions"

(************************************************************************
 * "name: value" entries, with continuations in the style of RFC 822;
 * "name=value" is also accepted
 * leading whitespace is removed from values
 *************************************************************************)
let entry = IniFile.entry setting sep comment


(************************************************************************
 * sections, led by a "[section]" header
 * We can't use titles as node names here since they could contain "/"
 * We remove #comment from possible keys
 * since it is used as label for comments
 * We also remove / as first character
 * because augeas doesn't like '/' keys (although it is legal in INI Files)
 *************************************************************************)
let title   = IniFile.title_label "target" IniFile.record_label_re
let record  = IniFile.record title entry

let lns    = IniFile.lns record comment

let filter = (incl "/etc/dput.cf")
           . (incl (Sys.getenv("HOME") . "/.dput.cf"))

let xfm = transform lns filter

