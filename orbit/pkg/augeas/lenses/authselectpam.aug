(*
Module: AuthselectPam
  Parses /etc/authselect/custom/*/*-auth and
  /etc/authselect/custom/*/postlogin files

Author: Heston Snodgrass <heston.snodgrass@puppet.com> based on pam.aug by David Lutterkort <lutter@redhat.com>

About: Reference
  This lens tries to keep as close as possible to `man pam.conf` where
  possible. This lens supports authselect templating syntax as
  can be found in `man authselect-profiles`.

About: Licence
  This file is licensed under the LGPL v2+, like the rest of Augeas.

About: Lens Usage

About: Configuration files
  This lens also autoloads /etc/authselect/custom/*/*-auth and
  /etc/authselect/custom/*/postlogin because these files are PAM template
  files on machines that have authselect custom profiles.
*)
module AuthselectPam =
  autoload xfm

  (* The Pam space does not work for certain parts of the authselect syntax so we need our own whitespace *)
  let reg_ws = del /([ \t])/ " "

  (* This is close the the same as argument from pam.aug, but curly braces are accounted for *)
  let argument = /(\[[^]{}#\n]+\]|[^[{#\n \t\\][^#\n \t\\]*)/

  (* The various types of conditional statements that can exist in authselect PAM files *)
  let authselect_conditional_type = /(continue if|stop if|include if|exclude if|imply|if)/

  (* Basic logical operators supported by authselect templates *)
  let authselect_logic_stmt = [ reg_ws . key /(and|or|not)/ ]

  (* authselect features inside conditional templates *)
  let authselect_feature = [ label "feature" . Quote.do_dquote (store /([a-z0-9-]+)/) ]

  (* authselect templates can substitute text if a condition is met. *)
  (* The sytax for this is `<conditional>:<what to sub on true>|<what to sub on false>` *)
  (* Both result forms are optional *)
  let authselect_on_true = [ label "on_true" . Util.del_str ":" . store /([^#{}:|\n\\]+)/ ]
  let authselect_on_false = [ label "on_false" . Util.del_str "|" . store /([^#{}:|\n\\]+)/ ]

  (* Features in conditionals can be grouped together so that logical operations can be resolved for the entire group *)
  let authselect_feature_group = [ label "feature_group" . Util.del_str "(" .
                                   authselect_feature . authselect_logic_stmt .
                                   reg_ws . authselect_feature . (authselect_logic_stmt . reg_ws . authselect_feature)* .
                                   Util.del_str ")" ]

  (* Represents a single, full authselect conditional template *)
  let authselect_conditional = [ Pam.space .
                                 Util.del_str "{" .
                                 label "authselect_conditional" . store authselect_conditional_type .
                                 authselect_logic_stmt* .
                                 ( reg_ws . authselect_feature | reg_ws . authselect_feature_group) .
                                 authselect_on_true? .
                                 authselect_on_false? .
                                 Util.del_str "}" ]

  (* Shared with PamConf *)
  let record = [ label "optional" . del "-" "-" ]? .
               [ label "type" . store Pam.types ] .
               Pam.space .
               [ label "control" . store Pam.control] .
               Pam.space .
               [ label "module" . store Pam.word ] .
               (authselect_conditional | [ Pam.space . label "argument" . store argument ])* .
               Pam.comment_or_eol

  let record_svc = [ seq "record" . Pam.indent . record ]

  let lns = ( Pam.empty | Pam.comment | Pam.include | record_svc ) *

  let filter = incl "/etc/authselect/custom/*/*-auth"
             . incl "/etc/authselect/custom/*/postlogin"
             . Util.stdexcl

  let xfm = transform lns filter

(* Local Variables: *)
(* mode: caml       *)
(* End:             *)
