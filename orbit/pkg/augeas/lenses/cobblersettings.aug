(*
    Parse the /etc/cobbler/settings file  which is in
    YAML 1.0 format.

    The lens can handle the following constructs
    * key: value
    * key: "value"
    * key: 'value'
    * key: [value1, value2]
    * key:
       - value1
       - value2
    * key:
       key2: value1
       key3: value2

    Author: Bryan Kearney

    About: License
      This file is licensed under the LGPL v2+, like the rest of Augeas.
*)
module CobblerSettings =
    autoload xfm

    let kw = /[a-zA-Z0-9_]+/
    (* TODO Would be better if this stripped off the "" and '' characters *)
    let kv = /([^]['", \t\n#:@-]+|"[^"\n]*"|'[^'\n]*')/

    let lbr = del /\[/ "["
    let rbr = del /\]/ "]"
    let colon = del /[ \t]*:[ \t]*/ ": "
    let dash = del /-[ \t]*/ "- "
    (* let comma = del /,[ \t]*(\n[ \t]+)?/ ", " *)
    let comma = del /[ \t]*,[ \t]*/ ", "

    let eol_only = del /\n/ "\n"

    (* TODO Would be better to make items a child of a document *)
    let docmarker = /-{3}/

    let eol   = Util.eol
    let comment = Util.comment
    let empty   = Util.empty
    let indent = del /[ \t]+/ "\t"
    let ws = del /[ \t]*/ " "

    let value_list = Build.opt_list [label "item" . store kv] comma
    let setting = [key kw . colon . store kv] . eol
    let simple_setting_suffix = store kv . eol
    let setting_list_suffix =  [label "sequence" . lbr . ws . (value_list . ws)? . rbr ] . eol
    let indendented_setting_list_suffix =  eol_only . (indent . setting)+
    let indented_list_suffix =  [label "list" . eol_only . ([ label "value" . indent . dash  . store kv] . eol)+]

    (* Break out setting because of a current bug in augeas *)
    let nested_setting = [key kw . colon . (
                                            (* simple_setting_suffix | *)
                                            setting_list_suffix |
                                            indendented_setting_list_suffix |
                                            indented_list_suffix
                                            )
                        ]

    let document = [label "---" . store docmarker] . eol

    let lns = (document | comment | empty | setting | nested_setting )*

    let xfm = transform lns (incl "/etc/cobbler/settings")


(* Local Variables: *)
(* mode: caml *)
(* End: *)
