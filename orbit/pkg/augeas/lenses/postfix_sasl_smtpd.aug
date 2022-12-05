module Postfix_sasl_smtpd =
  autoload xfm

  let eol = Util.eol
  let colon        = del /:[ \t]*/ ": "
  let value_to_eol = store Rx.space_in

  let simple_entry (kw:string) = [ key kw . colon . value_to_eol . eol ]

  let entries = simple_entry "pwcheck_method"
              | simple_entry "auxprop_plugin"
              | simple_entry "saslauthd_path"
              | simple_entry "mech_list"
              | simple_entry "sql_engine"
              | simple_entry "log_level"
              | simple_entry "auto_transition"

  let lns = entries+

  let filter = incl "/etc/postfix/sasl/smtpd.conf"
             . incl "/usr/local/etc/postfix/sasl/smtpd.conf"

  let xfm = transform lns filter
