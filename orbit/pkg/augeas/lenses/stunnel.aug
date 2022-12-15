(* Stunnel configuration file module for Augeas *)

module Stunnel =
    autoload xfm

    let comment = IniFile.comment IniFile.comment_re IniFile.comment_default
    let sep     = IniFile.sep "=" "="

    let setting = "chroot"
                | "compression"
                | "debug"
                | "EGD"
                | "engine"
                | "engineCtrl"
                | "fips"
                | "foreground"
                | "output"
                | "pid"
                | "RNDbytes"
                | "RNDfile"
                | "RNDoverwrite"
                | "service"
                | "setgid"
                | "setuid"
                | "socket"
                | "syslog"
                | "taskbar"
                | "accept"
                | "CApath"
                | "CAfile"
                | "cert"
                | "ciphers"
                | "client"
                | "connect"
                | "CRLpath"
                | "CRLfile"
                | "curve"
                | "delay"
                | "engineNum"
                | "exec"
                | "execargs"
                | "failover"
                | "ident"
                | "key"
                | "local"
                | "OCSP"
                | "OCSPflag"
                | "options"
                | "protocol"
                | "protocolAuthentication"
                | "protocolHost"
                | "protocolPassword"
                | "protocolUsername"
                | "pty"
                | "retry"
                | "session"
                | "sessiond"
                | "sni"
                | "sslVersion"
                | "stack"
                | "TIMEOUTbusy"
                | "TIMEOUTclose"
                | "TIMEOUTconnect"
                | "TIMEOUTidle"
                | "transparent"
                | "verify"

    let entry   = IniFile.indented_entry setting sep comment
    let empty   = IniFile.empty

    let title   = IniFile.indented_title ( IniFile.record_re - ".anon" )
    let record  = IniFile.record title entry

    let rc_anon = [ label ".anon" . ( entry | empty )+ ]

    let lns     = rc_anon? . record*

    let filter  = (incl "/etc/stunnel/stunnel.conf")

    let xfm     = transform lns filter
