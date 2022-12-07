(*
Module: Htpasswd
    Parses htpasswd and rsyncd.secrets files

Author: Marc Fournier <marc.fournier@camptocamp.com>

About: Reference
    This lens is based on examples in htpasswd(1) and rsyncd.conf(5)

About: Usage Example
(start code)
    augtool> set /augeas/load/Htpasswd/lens "Htpasswd.lns"
    augtool> set /augeas/load/Htpasswd/incl "/var/www/.htpasswd"
    augtool> load

    augtool> get /files/var/www/.htpasswd/foo
    /files/var/www/.htpasswd/foo = $apr1$e2WS6ARQ$lYhqy9CLmwlxR/07TLR46.

    augtool> set /files/var/www/.htpasswd/foo bar
    augtool> save
    Saved 1 file(s)

    $ cat /var/www/.htpasswd
    foo:bar
(end code)

About: License
    This file is licensed under the LGPL v2+, like the rest of Augeas.
*)

module Htpasswd =
autoload xfm

let entry = Build.key_value_line Rx.word Sep.colon (store Rx.space_in)
let lns   = (Util.empty | Util.comment | entry)*

let filter = incl "/etc/httpd/htpasswd"
           . incl "/etc/apache2/htpasswd"
           . incl "/etc/rsyncd.secrets"

let xfm = transform lns filter
