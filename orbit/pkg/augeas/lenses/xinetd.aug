(*
 * Module: Xinetd
 *   Parses xinetd configuration files
 *
 *  The structure of the lens and allowed attributes are ripped directly
 *  from xinetd's parser in xinetd/parse.c in xinetd's source checkout
 *  The downside of being so precise here is that if attributes are added
 *  they need to be added here, too. Writing a catchall entry, and getting
 *  to typecheck correctly would be a huge pain.
 *
 *  A really enterprising soul could tighten this down even further by
 *  restricting the acceptable values for each attribute.
 *
 * Author: David Lutterkort
 *)

module Xinetd =
  autoload xfm

  let opt_spc = Util.del_opt_ws " "

  let spc_equal = opt_spc . Sep.equal

  let op = ([ label "add" . opt_spc . Util.del_str "+=" ]
           |[ label "del" . opt_spc . Util.del_str "-=" ]
           | spc_equal)

  let value = store Rx.no_spaces

  let indent = del Rx.opt_space "\t"

  let attr_one (n:regexp) =
    Build.key_value n Sep.space_equal value

  let attr_lst (n:regexp) (op_eq: lens) =
    let value_entry =  [ label "value" . value ] in
    Build.key_value n op_eq (opt_spc . Build.opt_list value_entry Sep.space)?

  let attr_lst_eq (n:regexp) = attr_lst n spc_equal

  let attr_lst_op (n:regexp) = attr_lst n op

  (* Variable: service_attr
   *   Note:
   *      It is much faster to combine, for example, all the attr_one
   *      attributes into one regexp and pass that to a lens instead of
   *      using lens union (attr_one "a" | attr_one "b"|..) because the latter
   *      causes the type checker to work _very_ hard.
   *)
  let service_attr =
   attr_one (/socket_type|protocol|wait|user|group|server|instances/i
     |/rpc_version|rpc_number|id|port|nice|banner|bind|interface/i
     |/per_source|groups|banner_success|banner_fail|disable|max_load/i
     |/rlimit_as|rlimit_cpu|rlimit_data|rlimit_rss|rlimit_stack|v6only/i
     |/deny_time|umask|mdns|libwrap/i)
   (* redirect and cps aren't really lists, they take exactly two values *)
   |attr_lst_eq (/server_args|log_type|access_times|type|flags|redirect|cps/i)
   |attr_lst_op (/log_on_success|log_on_failure|only_from|no_access|env|passenv/i)

  let default_attr =
    attr_one (/instances|banner|bind|interface|per_source|groups/i
      |/banner_success|banner_fail|max_load|v6only|umask|mdns/i)
   |attr_lst_eq /cps/i       (* really only two values, not a whole list *)
   |attr_lst_op (/log_type|log_on_success|log_on_failure|disabled/i
      |/no_access|only_from|passenv|enabled/i)

  (* View: body
   *   Note:
   *       We would really like to say "the body can contain any of a list
   *       of a list of attributes, each of them at most once"; but that
   *       would require that we build a lens that matches the permutation
   *       of all attributes; with around 40 individual attributes, that's
   *       not computationally feasible, even if we didn't have to worry
   *       about how to write that down. The resulting regular expressions
   *       would simply be prohibitively large.
   *)
  let body (attr:lens) = Build.block_newlines_spc
                            (indent . attr . Util.eol)
                            Util.comment

  (* View: includes
   *  Note:
   *   It would be nice if we could use the directories given in include and
   *   includedir directives to parse additional files instead of hardcoding
   *   all the places where xinetd config files can be found; but that is
   *   currently not possible, and implementing that has a good amount of
   *   hairy corner cases to consider.
   *)
  let includes =
     Build.key_value_line /include(dir)?/ Sep.space (store Rx.no_spaces)

  let service =
     let sto_re = /[^# \t\n\/]+/ in
     Build.key_value_line "service" Sep.space (store sto_re . body service_attr)

  let defaults = [ key "defaults" . body default_attr . Util.eol ]

  let lns = ( Util.empty | Util.comment | includes | defaults | service )*

  let filter = incl "/etc/xinetd.d/*"
             . incl "/etc/xinetd.conf"
             . Util.stdexcl

  let xfm = transform lns filter

(* Local Variables: *)
(* mode: caml       *)
(* End:             *)
