(*
Module: OpenShift_Quickstarts
  Parses
    - /etc/openshift/quickstarts.json

Author: Brian Redbeard <redbeard@dead-city.org>

About: License
   This file is licenced under the LGPL v2+, conforming to the other components
   of Augeas.

About: Lens Usage
  Sample usage of this lens in augtool:

    * Get your current setup
      > print /files/etc/openshift/quickstarts.json
      ...

    * Delete the quickstart named WordPress
      > rm /files/etc/openshift/quickstarts.json/array/dict[*]/entry/dict/entry[*][string = 'WordPress']/../../../

  Saving your file:

      > save

About: Configuration files
        /etc/openshift/quickstarts.json - Quickstarts available via the
            OpenShift Console.

About: Examples
   The <Test_OpenShift_Quickstarts> file contains various examples and tests.
*)
module OpenShift_Quickstarts =
    autoload xfm

(* View: lns *)
let lns = Json.lns

(* Variable: filter *)
let filter = incl "/etc/openshift/quickstarts.json"

let xfm = transform lns filter
(* vim: set ts=4  expandtab  sw=4: *)
