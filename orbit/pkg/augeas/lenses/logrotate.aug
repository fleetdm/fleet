(* Logrotate module for Augeas                *)
(* Author: Raphael Pinson <raphink@gmail.com> *)
(* Patches from:                              *)
(*   Sean Millichamp <sean@bruenor.org>       *)
(*                                            *)
(* Supported :                                *)
(*   - defaults                               *)
(*   - rules                                  *)
(*   - (pre|post)rotate entries               *)
(*                                            *)
(* Todo :                                     *)
(*                                            *)

module Logrotate =
   autoload xfm

   let sep_spc = Sep.space
   let sep_val = del /[ \t]*=[ \t]*|[ \t]+/ " "
   let eol = Util.eol
   let num = Rx.relinteger
   let word = /[^,#= \n\t{}]+/
   let filename = Quote.do_quote_opt (store /\/[^"',#= \n\t{}]+/)
   let size = num . /[kMG]?/

   let indent = del Rx.opt_space "\t"

   (* define omments and empty lines *)
   let comment = Util.comment
   let empty   = Util.empty


   (* Useful functions *)

   let list_item = [ sep_spc . key /[^\/+,# \n\t{}]+/ ]
   let select_to_eol (kw:string) (select:regexp) = [ label kw . store select ]
   let value_to_eol (kw:string) (value:regexp)  = Build.key_value kw sep_val (store value)
   let flag_to_eol (kw:string) = Build.flag kw
   let list_to_eol (kw:string) = [ key kw . list_item+ ]


   (* Defaults *)

   let create =
     let mode = sep_spc . [ label "mode" . store num ] in
     let owner = sep_spc . [ label "owner" . store word ] in
     let group = sep_spc . [ label "group" . store word ] in
     [ key "create" .
         ( mode | mode . owner | mode . owner . group )? ]

   let su =
     let owner = sep_spc . [ label "owner" . store word ] in
     let group = sep_spc . [ label "group" . store word ] in
     [ key "su" .
         ( owner | owner . group )? ]

   let tabooext = [ key "tabooext" . ( sep_spc . store /\+/ )? . list_item+ ]

   let attrs = select_to_eol "schedule" /(hourly|daily|weekly|monthly|yearly)/
                | value_to_eol "rotate" num
        | create
        | flag_to_eol "nocreate"
        | su
        | value_to_eol "include" word
        | select_to_eol "missingok" /(no)?missingok/
        | select_to_eol "compress" /(no)?compress/
        | select_to_eol "delaycompress" /(no)?delaycompress/
        | select_to_eol "ifempty" /(not)?ifempty/
        | select_to_eol "sharedscripts" /(no)?sharedscripts/
        | value_to_eol "size" size
        | tabooext
        | value_to_eol "olddir" word
        | flag_to_eol "noolddir"
        | value_to_eol "mail" word
        | flag_to_eol "mailfirst"
        | flag_to_eol "maillast"
        | flag_to_eol "nomail"
        | value_to_eol "errors" word
        | value_to_eol "extension" word
        | select_to_eol "dateext" /(no)?dateext/
        | value_to_eol "dateformat" word
        | flag_to_eol "dateyesterday"
        | value_to_eol "compresscmd" word
        | value_to_eol "uncompresscmd" word
        | value_to_eol "compressext" word
        | list_to_eol "compressoptions"
        | select_to_eol "copy" /(no)?copy/
        | select_to_eol "copytruncate" /(no)?copytruncate/
        | value_to_eol "maxage" num
        | value_to_eol "minsize" size
        | value_to_eol "maxsize" size
        | select_to_eol "shred" /(no)?shred/
        | value_to_eol "shredcycles" num
        | value_to_eol "start" num

   (* Define hooks *)


   let hook_lines =
     let line_re = /.*/ - /[ \t]*endscript[ \t]*/ in
       store ( line_re . ("\n" . line_re)* )? . Util.del_str "\n"

   let hooks =
     let hook_names = /(pre|post)rotate|(first|last)action/ in
     [ key hook_names . eol .
       hook_lines? .
       del /[ \t]*endscript/ "\tendscript" ]

   (* Define rule *)

   let body = Build.block_newlines
                 (indent . (attrs | hooks) . eol)
                 Util.comment

   let rule =
     let filename_entry = [ label "file" . filename ] in
     let filename_sep = del /[ \t\n]+/ " " in
     let filenames = Build.opt_list filename_entry filename_sep in
     [ label "rule" . Util.indent . filenames . body . eol ]

   let lns = ( comment | empty | (attrs . eol) | rule )*

   let filter = incl "/etc/logrotate.d/*"
              . incl "/etc/logrotate.conf"
          . Util.stdexcl

   let xfm = transform lns filter
