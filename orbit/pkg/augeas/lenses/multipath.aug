(* Process /etc/multipath.conf                             *)
(* The lens is based on the multipath.conf(5) man page     *)
module Multipath =

autoload xfm

let comment = Util.comment
let empty = Util.empty
let dels = Util.del_str
let eol = Util.eol

let ws = del /[ \t]+/ " "
let indent = del /[ \t]*/ ""
(* We require that braces are always followed by a newline *)
let obr = del /\{([ \t]*)\n/ "{\n"
let cbr = del /[ \t]*}[ \t]*\n/ "}\n"

(* Like Rx.fspath, but we disallow quotes at the beginning or end *)
let fspath = /[^" \t\n]|[^" \t\n][^ \t\n]*[^" \t\n]/

let ikey (k:regexp) = indent . key k

let section (n:regexp) (b:lens) =
  [ ikey n . ws . obr . (b|empty|comment)* . cbr ]

let kv (k:regexp) (v:regexp) =
  [ ikey k . ws . del /"?/ "" . store v . del /"?/ "" . eol ]

(* FIXME: it would be much more concise to write                       *)
(* [ key k . ws . (bare | quoted) ]                                    *)
(* but the typechecker trips over that                                 *)
let qstr (k:regexp) =
  let delq = del /['"]/ "\"" in
  let bare = del /["']?/ "" . store /[^"' \t\n]+/ . del /["']?/ "" in
  let quoted = delq . store /.*[ \t].*/ . delq in
  [ ikey k . ws . bare . eol ]
 |[ ikey k . ws . quoted . eol ]

(* Settings that can be changed in various places *)
let common_setting =
   qstr "path_selector"
  |kv "path_grouping_policy" /failover|multibus|group_by_(serial|prio|node_name)/
  |kv "path_checker" /tur|emc_clariion|hp_sw|rdac|directio|rdb|readsector0/
  |kv "prio" /const|emc|alua|ontap|rdac|hp_sw|hds|random|weightedpath/
  |qstr "prio_args"
  |kv "failback" (Rx.integer | /immediate|manual|followover/)
  |kv "rr_weight" /priorities|uniform/
  |kv "flush_on_last_del" /yes|no/
  |kv "user_friendly_names" /yes|no/
  |kv "no_path_retry" (Rx.integer | /fail|queue/)
  |kv /rr_min_io(_q)?/ Rx.integer
  |qstr "features"
  |kv "reservation_key" Rx.word
  |kv "deferred_remove" /yes|no/
  |kv "delay_watch_checks" (Rx.integer | "no")
  |kv "delay_wait_checks" (Rx.integer | "no")
  |kv "skip_kpartx" /yes|no/
  (* Deprecated settings for backwards compatibility *)
  |qstr /(getuid|prio)_callout/
  (* Settings not documented in `man multipath.conf` *)
  |kv /rr_min_io_rq/ Rx.integer
  |kv "udev_dir" fspath
  |qstr "selector"
  |kv "async_timeout" Rx.integer
  |kv "pg_timeout" Rx.word
  |kv "h_on_last_deleassign_maps" /yes|no/
  |qstr "uid_attribute"
  |kv "hwtable_regex_match" /yes|no|on|off/
  |kv "reload_readwrite" /yes|no/

let default_setting =
   common_setting
  |kv "polling_interval" Rx.integer
  |kv "max_polling_interval" Rx.integer
  |kv "multipath_dir" fspath
  |kv "find_multipaths" /yes|no/
  |kv "verbosity" /[0-6]/
  |kv "reassign_maps" /yes|no/
  |kv "uid_attrribute" Rx.word
  |kv "max_fds" (Rx.integer|"max")
  |kv "checker_timeout" Rx.integer
  |kv "fast_io_fail_tmo" (Rx.integer|"off")
  |kv "dev_loss_tmo" (Rx.integer|"infinity")
  |kv "queue_without_daemon" /yes|no/
  |kv "bindings_file" fspath
  |kv "wwids_file" fspath
  |kv "log_checker_err" /once|always/
  |kv "retain_attached_hw_handler" /yes|no/
  |kv "detect_prio" /yes|no/
  |kv "hw_str_match" /yes|no/
  |kv "force_sync" /yes|no/
  |kv "config_dir" fspath
  |kv "missing_uev_wait_timeout" Rx.integer
  |kv "ignore_new_boot_devs" /yes|no/
  |kv "retrigger_tries" Rx.integer
  |kv "retrigger_delay" Rx.integer
  |kv "new_bindings_in_boot" /yes|no/

(* A device subsection *)
let device =
  let setting =
    qstr /vendor|product|product_blacklist|hardware_handler|alias_prefix/
   |default_setting in
  section "device" setting

(* The defaults section *)
let defaults =
  section "defaults" default_setting

(* The blacklist and blacklist_exceptions sections *)
let blacklist =
  let setting =
    qstr /devnode|wwid|property/
   |device in
  section /blacklist(_exceptions)?/ setting

(* A multipath subsection *)
let multipath =
  let setting =
    kv "wwid" (Rx.word|"*")
   |qstr "alias"
   |common_setting in
  section "multipath" setting

(* The multipaths section *)
let multipaths =
  section "multipaths" multipath

(* The devices section *)
let devices =
  section "devices" device

let lns = (comment|empty|defaults|blacklist|devices|multipaths)*

let xfm = transform lns (incl "/etc/multipath.conf" .
                         incl "/etc/multipath/conf.d/*.conf")
