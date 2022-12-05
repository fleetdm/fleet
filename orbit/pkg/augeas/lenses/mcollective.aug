(*
Module: MCollective
  Parses MCollective's configuration files

Author: Marc Fournier <marc.fournier@camptocamp.com>

About: Reference
    This lens is based on MCollective's default client.cfg and server.cfg.

About: Usage Example
(start code)
    augtool> get /files/etc/mcollective/client.cfg/plugin.psk
    /files/etc/mcollective/client.cfg/plugin.psk = unset

    augtool> ls /files/etc/mcollective/client.cfg/
    topicprefix = /topic/
    main_collective = mcollective
    collectives = mcollective
    [...]

    augtool> set /files/etc/mcollective/client.cfg/plugin.stomp.password example123
    augtool> save
    Saved 1 file(s)
(end code)
   The <Test_MCollective> file also contains various examples.

About: License
  This file is licensed under the LGPL v2+, like the rest of Augeas.
*)

module MCollective =
autoload xfm

let lns = Simplevars.lns

let filter = incl "/etc/mcollective/client.cfg"
           . incl "/etc/mcollective/server.cfg"
           . incl "/etc/puppetlabs/mcollective/client.cfg"
           . incl "/etc/puppetlabs/mcollective/server.cfg"

let xfm = transform lns filter
