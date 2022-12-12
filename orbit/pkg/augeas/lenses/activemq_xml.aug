(*
Module: ActiveMQ_XML
  ActiveMQ / FuseMQ XML module for Augeas

Author: Brian Redbeard <redbeard@dead-city.org>

About: Reference
  This lens ensures that XML files included in ActiveMQ / FuseMQ are properly
  handled by Augeas.

About: License
  This file is licensed under the LGPL License.

About: Lens Usage
  Sample usage of this lens in augtool:

    * Get your current setup
      > print /files/etc/activemq/activemq.xml
      ...

    * Change OpenShift domain
      > set /files/etc/openshift/broker.conf/CLOUD_DOMAIN ose.example.com

  Saving your file:

      > save

About: Configuration files
  This lens applies to relevant XML files located in  /etc/activemq/ . See <filter>.

*)

module ActiveMQ_XML =
        autoload xfm

let lns = Xml.lns

let filter = (incl "/etc/activemq/*.xml") 
           . Util.stdexcl

let xfm = transform lns filter
