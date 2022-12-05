module DevfsRules =

  autoload xfm

  let comment  = IniFile.comment IniFile.comment_re "#"

  let eol = Util.eol

  let line_re = /[^][#; \t\n][^#;\n]*[^#; \t\n]/
  let entry = [ seq "entry" . store line_re . (eol | comment) ]

  let title = Util.del_str "["
            . key Rx.word . [ label "id" . Sep.equal . store Rx.integer ]
            . Util.del_str "]" . eol
            . counter "entry"

  let record = IniFile.record title (entry | comment)

  let lns = IniFile.lns record comment

  let filter = incl "/etc/defaults/devfs.rules"
            .  incl "/etc/devfs.rules"

  let xfm = transform lns filter
