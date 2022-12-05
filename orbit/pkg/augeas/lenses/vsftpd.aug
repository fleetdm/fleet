(* Parse vsftpd.conf *)
module Vsftpd =
  autoload xfm

(* The code in parseconf.c does not seem to allow for trailing whitespace *)
(* in the config file                                                     *)
let eol = Util.del_str "\n"
let empty = Util.empty
let comment = Util.comment

let bool_option_re = /anonymous_enable|isolate|isolate_network|local_enable|pasv_enable|port_enable|chroot_local_user|write_enable|anon_upload_enable|anon_mkdir_write_enable|anon_other_write_enable|chown_uploads|connect_from_port_20|xferlog_enable|dirmessage_enable|anon_world_readable_only|async_abor_enable|ascii_upload_enable|ascii_download_enable|one_process_model|xferlog_std_format|pasv_promiscuous|deny_email_enable|chroot_list_enable|setproctitle_enable|text_userdb_names|ls_recurse_enable|log_ftp_protocol|guest_enable|userlist_enable|userlist_deny|use_localtime|check_shell|hide_ids|listen|port_promiscuous|passwd_chroot_enable|no_anon_password|tcp_wrappers|use_sendfile|force_dot_files|listen_ipv6|dual_log_enable|syslog_enable|background|virtual_use_local_privs|session_support|download_enable|dirlist_enable|chmod_enable|secure_email_list_enable|run_as_launching_user|no_log_lock|ssl_enable|allow_anon_ssl|force_local_logins_ssl|force_local_data_ssl|ssl_sslv2|ssl_sslv3|ssl_tlsv1|tilde_user_enable|force_anon_logins_ssl|force_anon_data_ssl|mdtm_write|lock_upload_files|pasv_addr_resolve|debug_ssl|require_cert|validate_cert|require_ssl_reuse|allow_writeable_chroot|seccomp_sandbox/

let uint_option_re = /accept_timeout|connect_timeout|local_umask|anon_umask|ftp_data_port|idle_session_timeout|data_connection_timeout|pasv_min_port|pasv_max_port|anon_max_rate|local_max_rate|listen_port|max_clients|file_open_mode|max_per_ip|trans_chunk_size|delay_failed_login|delay_successful_login|max_login_fails|chown_upload_mode/

let str_option_re = /secure_chroot_dir|ftp_username|chown_username|xferlog_file|vsftpd_log_file|message_file|nopriv_user|ftpd_banner|banned_email_file|chroot_list_file|pam_service_name|guest_username|userlist_file|anon_root|local_root|banner_file|pasv_address|listen_address|user_config_dir|listen_address6|cmds_allowed|hide_file|deny_file|user_sub_token|email_password_file|rsa_cert_file|dsa_cert_file|ssl_ciphers|rsa_private_key_file|dsa_private_key_file|ca_certs_file/

let bool_value_re = /[yY][eE][sS]|[tT][rR][uU][eE]|1|[nN][oO]|[fF][aA][lL][sS][eE]|0/

let option (k:regexp) (v:regexp) = [ key k . Util.del_str "=" . store v . eol ]

let bool_option = option bool_option_re bool_value_re

let str_option = option str_option_re /[^\n]+/

let uint_option = option uint_option_re /[0-9]+/

let lns = (bool_option|str_option|uint_option|comment|empty)*

let filter = (incl "/etc/vsftpd.conf") . (incl "/etc/vsftpd/vsftpd.conf")

let xfm = transform lns filter
