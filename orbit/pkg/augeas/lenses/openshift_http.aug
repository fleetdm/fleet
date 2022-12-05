(*
Module: OpenShift_Http
  Parses HTTPD related files specific to openshift

Author: Brian Redbeard <redbeard@dead-city.org>

About: License
   This file is licenced under the LGPL v2+, conforming to the other components
   of Augeas.

About: Lens Usage
  Sample usage of this lens in augtool:

    * Get your current setup
      > print /files/var/www/openshift

About: Examples
   The <Test_OpenShift_Http> file contains various examples and tests.
*)
module OpenShift_Http =
    autoload xfm

(* Variable: filter *)
let filter =  incl "/var/www/openshift/console/httpd/httpd.conf"
            . incl "/var/www/openshift/console/httpd/conf.d/*.conf"
            . incl "/var/www/openshift/broker/httpd/conf.d/*.conf"
            . incl "/var/www/openshift/broker/httpd/httpd.conf"
            . incl "/var/www/openshift/console/httpd/console.conf"
            . incl "/var/www/openshift/broker/httpd/broker.conf"
            . Util.stdexcl

(* View: lns *) 
let lns = Httpd.lns 

let xfm = transform lns filter
(* vim: set ts=4  expandtab  sw=4: *)
