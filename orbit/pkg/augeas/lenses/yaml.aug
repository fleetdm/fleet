(*
Module: Yaml
  Only valid for the following subset:

> defaults: &anchor
>   repo1: master
>
> host:
>   # Inheritance
>   <<: *anchor
>   repo2: branch

Author: Dimitar Dimitrov <mitkofr@yahoo.fr>
*)
module YAML =

(* Group: helpers *)
let dash = Util.del_str "-"
let colon = Sep.colon
let space = Sep.space
let val = store Rx.word
let eol = Util.eol
let empty = Util.empty
let comment = Util.comment_noindent

(*
View: indent
  the imposed indent is 2 spaces
*)
let indent = del /[ \t]+/ "  "

let mval = [ label "@mval" . Util.del_str "|-" . eol
           . [ label "@line" . indent . store Rx.space_in . eol ]+ ]

(*
View: inherit
> <<: *anchor
*)
let _inherit = [ key "<<" . colon . space . Util.del_str "*" . val . eol ]
let inherit = indent . _inherit . (indent . comment)*

(*
View: repo
> { "repo" = "branch" }
*)
let _repo = [ key Rx.word . colon . space . (val | mval) . eol ]
let repo = indent . _repo . (indent . comment)*

(*
View: anchor
> &anchor
*)
let anchor = Util.del_str "&" . val

(*
View: entry
> host:
>   # Inheritance
>   <<: *anchor
>   repo2: branch
*)
let entry = [ key Rx.word . colon . (space . anchor)? . eol
            . (indent . comment)*
            . ((inherit . (repo+)?) | repo+)
            ]

(* View: top level sequence *)
let sequence = [ label "@sequence" . counter "sequence" . dash . repo+ ]

(* View: header *)
let header = [ label "@yaml" . Util.del_str "---"
             . (Sep.space . store Rx.space_in)? . eol ]

(*
View: lns
  The yaml lens
*)
let lns = ((empty|comment)* . header)? . (sequence | entry | comment | empty)*
