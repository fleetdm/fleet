(* OpenVPN module for Augeas
 Author: Raphael Pinson <raphink@gmail.com>
 Author: Justin Akers <dafugg@gmail.com>

 Reference: http://openvpn.net/index.php/documentation/howto.html
 Reference: https://community.openvpn.net/openvpn/wiki/Openvpn23ManPage

 TODO: Inline file support
*)


module OpenVPN =
  autoload xfm

(************************************************************************
 *                           USEFUL PRIMITIVES
 *************************************************************************)

let eol    = Util.eol
let indent = Util.indent

(* Define separators *)
let sep    = Util.del_ws_spc

(* Define value regexps.
   Custom simplified ipv6 used instead of Rx.ipv6 as the augeas Travis instances
   are limited to 2GB of memory. Using 'ipv6_re = Rx.ipv6' consumes an extra
   2GB of memory and thus the test is OOM-killed.
*)
let ipv6_re = /[0-9A-Fa-f:]+/
let ipv4_re = Rx.ipv4
let ip_re  = ipv4_re|ipv6_re
let num_re = Rx.integer
let fn_re  = /[^#; \t\n][^#;\n]*[^#; \t\n]|[^#; \t\n]/
let fn_safe_re = /[^#; \t\r\n]+/
let an_re  = /[a-z][a-z0-9_-]*/
let hn_re  = Rx.hostname
let port_re = /[0-9]+/
let host_re = ip_re|hn_re
let proto_re = /(tcp|udp)/
let proto_ext_re = /(udp|tcp-client|tcp-server)/
let alg_re = /(none|[A-Za-z][A-Za-z0-9-]+)/
let ipv6_bits_re = ipv6_re . /\/[0-9]+/

(* Define store aliases *)
let ip     = store ip_re
let num    = store num_re
let filename = store fn_re
let filename_safe = store fn_safe_re
let hostname = store hn_re
let sto_to_dquote = store /[^"\n]+/   (* " Emacs, relax *)
let port = store port_re
let host = store host_re
let proto = store proto_re
let proto_ext = store proto_ext_re

(* define comments and empty lines *)
let comment = Util.comment_generic /[ \t]*[;#][ \t]*/ "# "
let comment_or_eol = eol | Util.comment_generic /[ \t]*[;#][ \t]*/ " # "

let empty   = Util.empty


(************************************************************************
 *                               SINGLE VALUES
 *
 *   - local => IP|hostname
 *   - port  => num
 *   - proto => udp|tcp-client|tcp-server
 *   - proto-force => udp|tcp
 *   - mode  => p2p|server
 *   - dev   => (tun|tap)\d*
 *   - dev-node => filename
 *   - ca    => filename
 *   - config => filename
 *   - cert  => filename
 *   - key   => filename
 *   - dh    => filename
 *   - ifconfig-pool-persist => filename
 *   - learn-address => filename
 *   - cipher => [A-Z0-9-]+
 *   - max-clients => num
 *   - user  => alphanum
 *   - group => alphanum
 *   - status => filename
 *   - log   => filename
 *   - log-append => filename
 *   - client-config-dir => filename
 *   - verb => num
 *   - mute => num
 *   - fragment => num
 *   - mssfix   => num
 *   - connect-retry num
 *   - connect-retry-max num
 *   - connect-timeout num
 *   - http-proxy-timeout num
 *   - max-routes num
 *   - ns-cert-type => "server"
 *   - resolv-retry => "infinite"
 *   - script-security => [0-3] (execve|system)?
 *   - ipchange => command
 *   - topology => type
 *************************************************************************)

let single_host = "local" | "tls-remote"
let single_ip   = "lladdr"
let single_ipv6_bits = "iroute-ipv6"
                     | "server-ipv6"
                     | "ifconfig-ipv6-pool"
let single_num = "port"
               | "max-clients"
               | "verb"
               | "mute"
               | "fragment"
               | "mssfix"
               | "connect-retry"
               | "connect-retry-max"
               | "connect-timeout"
               | "http-proxy-timeout"
               | "resolv-retry"
               | "lport"
               | "rport"
               | "max-routes"
               | "max-routes-per-client"
               | "route-metric"
               | "tun-mtu"
               | "tun-mtu-extra"
               | "shaper"
               | "ping"
               | "ping-exit"
               | "ping-restart"
               | "sndbuf"
               | "rcvbuf"
               | "txqueuelen"
               | "link-mtu"
               | "nice"
               | "management-log-cache"
               | "bcast-buffers"
               | "tcp-queue-limit"
               | "server-poll-timeout"
               | "keysize"
               | "pkcs11-pin-cache"
               | "tls-timeout"
               | "reneg-bytes"
               | "reneg-pkts"
               | "reneg-sec"
               | "hand-window"
               | "tran-window"
let single_fn   = "ca"
                | "cert"
                | "extra-certs"
                | "config"
                | "key"
                | "dh"
                | "log"
                | "log-append"
                | "client-config-dir"
                | "dev-node"
                | "cd"
                | "chroot"
                | "writepid"
                | "client-config-dir"
                | "tmp-dir"
                | "replay-persist"
                | "ca"
                | "capath"
                | "pkcs12"
                | "pkcs11-id"
                | "askpass"
                | "tls-export-cert"
                | "x509-track"
let single_an  = "user"
               | "group"
               | "management-client-user"
               | "management-client-group"
let single_cmd = "ipchange"
                | "iproute"
                | "route-up"
                | "route-pre-down"
                | "mark"
                | "up"
                | "down"
                | "setcon"
                | "echo"
                | "client-connect"
                | "client-disconnect"
                | "learn-address"
                | "tls-verify"

let single_entry (kw:regexp) (re:regexp)
               = [ key kw . sep . store re . comment_or_eol ]

let single_opt_entry (kw:regexp) (re:regexp)
                = [ key kw . (sep . store re)? .comment_or_eol ]

let single     = single_entry single_num num_re
      	       | single_entry single_fn  fn_re
	       | single_entry single_an  an_re
	       | single_entry single_host host_re
	       | single_entry single_ip ip_re
           | single_entry single_ipv6_bits ipv6_bits_re
           | single_entry single_cmd fn_re
	       | single_entry "proto"    proto_ext_re
	       | single_entry "proto-force"    proto_re
	       | single_entry "mode"    /(p2p|server)/
               | single_entry "dev"      /(tun|tap)[0-9]*|null/
	       | single_entry "dev-type"      /(tun|tap)/
	       | single_entry "topology"      /(net30|p2p|subnet)/
	       | single_entry "cipher" alg_re
	       | single_entry "auth" alg_re
	       | single_entry "resolv-retry" "infinite"
	       | single_entry "script-security" /[0-3]( execve| system)?/
	       | single_entry "route-gateway" (host_re|/dhcp/)
	       | single_entry "mtu-disc" /(no|maybe|yes)/
	       | single_entry "remap-usr1" /SIG(HUP|TERM)/
	       | single_entry "socket-flags" /(TCP_NODELAY)/
           | single_entry "auth-retry" /(none|nointeract|interact)/
           | single_entry "tls-version-max" Rx.decimal
           | single_entry "verify-hash" /([A-Za-z0-9]{2}:)+[A-Za-z0-9]{2}/
           | single_entry "pkcs11-cert-private" /[01]/
           | single_entry "pkcs11-protected-authentication" /[01]/
           | single_entry "pkcs11-private-mode" /[A-Za-z0-9]+/
           | single_entry "key-method" /[12]/
           | single_entry "ns-cert-type" /(client|server)/
           | single_entry "remote-cert-tls" /(client|server)/

let single_opt  = single_opt_entry "comp-lzo" /(yes|no|adaptive)/
                | single_opt_entry "syslog" fn_re
                | single_opt_entry "daemon" fn_re
                | single_opt_entry "auth-user-pass" fn_re
                | single_opt_entry "explicit-exit-notify" num_re
                | single_opt_entry "engine" fn_re

(************************************************************************
 *                               DOUBLE VALUES
 *************************************************************************)

let double_entry (kw:regexp) (a:string) (aval:regexp) (b:string) (bval:regexp)
    = [ key kw
      . sep . [ label a . store aval ]
      . sep . [ label b . store bval ]
      . comment_or_eol
      ]

let double_secopt_entry (kw:regexp) (a:string) (aval:regexp) (b:string) (bval:regexp)
    = [ key kw
      . sep . [ label a . store aval ]
      . (sep . [ label b . store bval ])?
      . comment_or_eol
      ]


let double  = double_entry "keepalive" "ping" num_re "timeout" num_re
            | double_entry "hash-size" "real" num_re "virtual" num_re
            | double_entry "ifconfig" "local" ip_re "remote" ip_re
            | double_entry "connect-freq" "num" num_re "sec" num_re
            | double_entry "verify-x509-name" "name" hn_re "type"
                /(subject|name|name-prefix)/
            | double_entry "ifconfig-ipv6" "address" ipv6_bits_re "remote" ipv6_re
            | double_entry "ifconfig-ipv6-push" "address" ipv6_bits_re "remote" ipv6_re
            | double_secopt_entry "iroute" "local" ip_re "netmask" ip_re
            | double_secopt_entry "stale-routes-check" "age" num_re "interval" num_re
            | double_secopt_entry "ifconfig-pool-persist"
                "file" fn_safe_re "seconds" num_re
            | double_secopt_entry "secret" "file" fn_safe_re "direction" /[01]/
            | double_secopt_entry "prng" "algorithm" alg_re "nsl" num_re
            | double_secopt_entry "replay-window" "window-size" num_re "seconds" num_re


(************************************************************************
 *                               FLAGS
 *************************************************************************)

let flag_words = "client-to-client"
               | "duplicate-cn"
	       | "persist-key"
	       | "persist-tun"
	       | "client"
	       | "remote-random"
	       | "nobind"
	       | "mute-replay-warnings"
	       | "http-proxy-retry"
	       | "socks-proxy-retry"
           | "remote-random-hostname"
           | "show-proxy-settings"
           | "float"
           | "bind"
           | "nobind"
           | "tun-ipv6"
           | "ifconfig-noexec"
           | "ifconfig-nowarn"
           | "route-noexec"
           | "route-nopull"
           | "allow-pull-fqdn"
           | "mtu-test"
           | "ping-timer-rem"
           | "persist-tun"
           | "persist-local-ip"
           | "persist-remote-ip"
           | "mlock"
           | "up-delay"
           | "down-pre"
           | "up-restart"
           | "disable-occ"
           | "errors-to-stderr"
           | "passtos"
           | "suppress-timestamps"
           | "fast-io"
           | "multihome"
           | "comp-noadapt"
           | "management-client"
           | "management-query-passwords"
           | "management-query-proxy"
           | "management-query-remote"
           | "management-forget-disconnect"
           | "management-hold"
           | "management-signal"
           | "management-up-down"
           | "management-client-auth"
           | "management-client-pf"
           | "push-reset"
           | "push-peer-info"
           | "disable"
           | "ifconfig-pool-linear"
           | "client-to-client"
           | "duplicate-cn"
           | "ccd-exclusive"
           | "tcp-nodelay"
           | "opt-verify"
           | "auth-user-pass-optional"
           | "client-cert-not-required"
           | "username-as-common-name"
           | "pull"
           | "key-direction"
           | "no-replay"
           | "mute-replay-warnings"
           | "no-iv"
           | "use-prediction-resistance"
           | "test-crypto"
           | "tls-server"
           | "tls-client"
           | "pkcs11-id-management"
           | "single-session"
           | "tls-exit"
           | "auth-nocache"
           | "show-ciphers"
           | "show-digests"
           | "show-tls"
           | "show-engines"
           | "genkey"
           | "mktun"
           | "rmtun"


let flag_entry (kw:regexp)
               = [ key kw . comment_or_eol ]

let flag       = flag_entry flag_words


(************************************************************************
 *                               OTHER FIELDS
 *
 *   - server        => IP IP [nopool]
 *   - server-bridge => IP IP IP IP
 *   - route	     => host host [host [num]]
 *   - push          => "string"
 *   - tls-auth      => filename [01]
 *   - remote        => hostname/IP [num] [(tcp|udp)]
 *   - management    => IP num filename
 *   - http-proxy    => host port [filename|keyword] [method]
 *   - http-proxy-option => (VERSION decimal|AGENT string)
 *   ...
 *   and many others
 *
 *************************************************************************)

let server          = [ key "server"
                      . sep . [ label "address" . ip ]
                      . sep . [ label "netmask" . ip ]
                      . (sep . [ key "nopool" ]) ?
                      . comment_or_eol
                      ]

let server_bridge =
    let ip_params = [ label "address" . ip ] . sep
        . [ label "netmask" . ip ] . sep
        . [ label "start"   . ip ] . sep
        . [ label "end"     . ip ] in
            [ key "server-bridge"
            . sep . (ip_params|store /(nogw)/)
            . comment_or_eol
            ]

let route =
    let route_net_kw   = store (/(vpn_gateway|net_gateway|remote_host)/|host_re) in
        [ key "route" . sep
        . [ label "address" . route_net_kw ]
        . (sep . [ label "netmask" . store (ip_re|/default/) ]
            . (sep . [ label "gateway" . route_net_kw ]
                . (sep . [ label "metric" . store (/default/|num_re)] )?
            )?
        )?
        . comment_or_eol
        ]

let route_ipv6 =
    let route_net_re = /(vpn_gateway|net_gateway|remote_host)/ in
        [ key "route-ipv6" . sep
        . [ label "network" . store (route_net_re|ipv6_bits_re) ]
        . (sep . [ label "gateway" . store (route_net_re|ipv6_re) ]
            . (sep . [ label "metric" . store (/default/|num_re)] )?
        )?
        . comment_or_eol
        ]

let push          = [ key "push" . sep
                    . Quote.do_dquote sto_to_dquote
		    . comment_or_eol
                    ]

let tls_auth      = [ key "tls-auth" . sep
                    . [ label "key"       . filename     ] . sep
		    . [ label "is_client" . store /[01]/ ] . comment_or_eol
                    ]

let remote        = [ key "remote" . sep
                    . [ label "server" . host ]
		            . (sep . [label "port" . port]
                        . (sep . [label "proto" . proto]) ? ) ?
                    . comment_or_eol
		    ]

let http_proxy =
    let auth_method_re = /(none|basic|ntlm)/ in
        let auth_method = store auth_method_re in
            [ key "http-proxy"
            . sep . [ label "server" . host ]
            . sep . [ label "port"   . port ]
            . (sep . [ label "auth" .  filename_safe ]
                . (sep . [ label "auth-method" . auth_method ]) ? )?
            . comment_or_eol
            ]

let http_proxy_option = [ key "http-proxy-option"
                        . sep . [ label "option" . store /(VERSION|AGENT)/ ]
                        . sep . [ label "value" . filename ]
                        . comment_or_eol
                        ]

let socks_proxy     = [ key "socks-proxy"
                      . sep . [ label "server" . host ]
                      . (sep . [ label "port"   . port ]
                        . (sep . [ label "auth" .  filename_safe ])? )?
                      . comment_or_eol
                      ]

let port_share      = [ key "port-share"
                      . sep . [ label "host" . host ]
                      . sep . [ label "port" . port ]
                      . (sep . [ label "dir" . filename ])?
                      . comment_or_eol
                      ]

let route_delay     = [ key "route-delay"
                    . (sep . [ label "seconds" . num ]
                        . (sep . [ label "win-seconds" . num ] ) ?
                    )?
                    . comment_or_eol
                    ]

let inetd           = [ key "inetd"
                    . (sep . [label "mode" . store /(wait|nowait)/ ]
                        . (sep . [ label "progname" . filename ] ) ?
                    )?
                    . comment_or_eol
                    ]

let inactive        = [ key "inactive"
                    . sep . [ label "seconds" . num ]
                    . (sep . [ label "bytes" . num ] ) ?
                    . comment_or_eol
                    ]

let client_nat      = [ key "client-nat"
                    . sep . [ label "type" . store /(snat|dnat)/ ]
                    . sep . [ label "network" . ip ]
                    . sep . [ label "netmask" . ip ]
                    . sep . [ label "alias" . ip ]
                    . comment_or_eol
                    ]

let status          = [ key "status"
                    . sep . [ label "file" . filename_safe ]
                    . (sep . [ label "repeat-seconds" . num ]) ?
                    . comment_or_eol
                    ]

let plugin          = [ key "plugin"
                    . sep . [ label "file" . filename_safe ]
                    . (sep . [ label "init-string" . filename ]) ?
                    . comment_or_eol
                    ]

let management    = [ key "management" . sep
                    . [ label "server" . ip ]
                    . sep . [ label "port" . port ]
                    . (sep . [ label "pwfile" . filename ] ) ?
                    . comment_or_eol
                    ]

let auth_user_pass_verify   = [ key "auth-user-pass-verify"
                              . sep . [ Quote.quote_spaces (label "command") ]
                              . sep . [ label "method" . store /via-(env|file)/ ]
                              . comment_or_eol
                              ]

let static_challenge    = [ key "static-challenge"
                          . sep . [ Quote.quote_spaces (label "text") ]
                          . sep . [ label "echo" . store /[01]/ ]
                          . comment_or_eol
                          ]

let cryptoapicert        = [ key "cryptoapicert" . sep . Quote.dquote
                          . [ key /[A-Z]+/ . Sep.colon . store /[A-Za-z _-]+/ ]
                          . Quote.dquote . comment_or_eol
                          ]

let setenv =
    let envvar = /[^#;\/ \t\n][A-Za-z0-9_-]+/ in
        [ key ("setenv"|"setenv-safe")
        . sep . [ key envvar . sep . store fn_re ]
        . comment_or_eol
        ]

let redirect =
    let redirect_flag   = /(local|autolocal|def1|bypass-dhcp|bypass-dns|block-local)/ in
        let redirect_key    = "redirect-gateway" | "redirect-private" in
            [ key redirect_key
            . (sep . [ label "flag" . store redirect_flag ] ) +
            . comment_or_eol
            ]

let tls_cipher =
    let ciphername = /[A-Za-z0-9!_-]+/ in
        [ key "tls-cipher" . sep
        . [label "cipher" . store ciphername]
        . (Sep.colon . [label "cipher" . store ciphername])*
        . comment_or_eol
        ]

let remote_cert_ku =
    let usage = [label "usage" . store /[A-Za-z0-9]{1,2}/] in
        [ key "remote-cert-ku" . sep . usage . (sep . usage)* . comment_or_eol ]

(* FIXME: Surely there's a nicer way to do this *)
let remote_cert_eku =
    let oid = [label "oid" . store /[0-9]+\.([0-9]+\.)*[0-9]+/] in
        let symbolic = [Quote.do_quote_opt
            (label "symbol" . store /[A-Za-z0-9][A-Za-z0-9 _-]*[A-Za-z0-9]/)] in
            [ key "remote-cert-eku" . sep . (oid|symbolic) . comment_or_eol ]

let status_version          = [ key "status-version"
                              . (sep . num) ?
                              . comment_or_eol
                              ]

let ifconfig_pool           = [ key "ifconfig-pool"
                              . sep . [ label "start" . ip ]
                              . sep . [ label "end" . ip ]
                              . (sep . [ label "netmask" . ip ])?
                              . comment_or_eol
                              ]

let ifconfig_push           = [ key "ifconfig-push"
                              . sep . [ label "local" . ip ]
                              . sep . [ label "remote-netmask" . ip ]
                              . (sep . [ label "alias" . store /[A-Za-z0-9_-]+/ ] )?
                              . comment_or_eol
                              ]

let ignore_unknown_option   = [ key "ignore-unknown-option"
                              . (sep . [ label "opt" . store /[A-Za-z0-9_-]+/ ] ) +
                              . comment_or_eol
                              ]

let tls_version_min         = [ key "tls-version-min"
                              . sep . store Rx.decimal
                              . (sep . [ key "or-highest" ]) ?
                              . comment_or_eol
                              ]

let crl_verify              = [ key "crl-verify"
                              . sep . filename_safe
                              . (sep . [ key "dir" ]) ?
                              . comment_or_eol
                              ]

let x509_username_field =
    let fieldname = /[A-Za-z0-9_-]+/ in
        let extfield = ([key /ext/ . Sep.colon . store fieldname]) in
            let subjfield = ([label "subj" . store fieldname]) in
                [ key "x509-username-field"
                . sep . (extfield|subjfield)
                . comment_or_eol
                ]

let other   = server
            | server_bridge
            | route
            | push
            | tls_auth
            | remote
            | http_proxy
            | http_proxy_option
            | socks_proxy
            | management
            | route_delay
            | client_nat
            | redirect
            | inactive
            | setenv
            | inetd
            | status
            | status_version
            | plugin
            | ifconfig_pool
            | ifconfig_push
            | ignore_unknown_option
            | auth_user_pass_verify
            | port_share
            | static_challenge
            | tls_version_min
            | tls_cipher
            | cryptoapicert
            | x509_username_field
            | remote_cert_ku
            | remote_cert_eku
            | crl_verify
            | route_ipv6


(************************************************************************
 *                              LENS & FILTER
 *************************************************************************)

let lns    = ( comment | empty | single | single_opt | double | flag | other )*

let filter = (incl "/etc/openvpn/client.conf")
           . (incl "/etc/openvpn/server.conf")

let xfm = transform lns filter



