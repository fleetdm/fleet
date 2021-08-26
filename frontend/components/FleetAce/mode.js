/* eslint-disable */
import { osqueryTableNames } from "utilities/osquery_tables";

ace.define(
  "ace/mode/fleet_highlight_rules",
  [
    "require",
    "exports",
    "module",
    "ace/lib/oop",
    "ace/mode/sql_highlight_rules",
  ],
  function (acequire, exports, module) {
    "use strict";

    var oop = acequire("../lib/oop");
    var SqlHighlightRules = acequire("./sql_highlight_rules").SqlHighlightRules;

    var FleetHighlightRules = function () {
      var keywords =
        "select|insert|update|delete|from|where|and|or|group|by|order|limit|offset|having|as|case|" +
        "when|else|end|type|left|right|join|on|outer|desc|asc|union|create|table|primary|key|if|" +
        "foreign|not|references|default|null|inner|cross|natural|database|drop|grant";

      var builtinConstants = "true|false";

      var builtinFunctions =
        "avg|count|first|last|max|min|sum|ucase|lcase|mid|len|round|rank|now|format|" +
        "coalesce|ifnull|isnull|nvl";

      var dataTypes =
        "int|numeric|decimal|date|varchar|char|bigint|float|double|bit|binary|text|set|timestamp|" +
        "money|real|number|integer";

      var osqueryTables = osqueryTableNames.join("|");

      var keywordMapper = this.createKeywordMapper(
        {
          "osquery-token": osqueryTables,
          "support.function": builtinFunctions,
          keyword: keywords,
          "constant.language": builtinConstants,
          "storage.type": dataTypes,
        },
        "identifier",
        true
      );

      this.$rules = {
        start: [
          {
            token: "comment",
            regex: "--.*$",
          },
          {
            token: "comment",
            start: "/\\*",
            end: "\\*/",
          },
          {
            token: "string", // " string
            regex: '".*?"',
          },
          {
            token: "string", // ' string
            regex: "'.*?'",
          },
          {
            token: "constant.numeric", // float
            regex: "[+-]?\\d+(?:(?:\\.\\d*)?(?:[eE][+-]?\\d+)?)?\\b",
          },
          {
            token: keywordMapper,
            regex: "[a-zA-Z_$][a-zA-Z0-9_$]*\\b",
          },
          {
            token: "keyword.operator",
            regex:
              "\\+|\\-|\\/|\\/\\/|%|<@>|@>|<@|&|\\^|~|<|>|<=|=>|==|!=|<>|=",
          },
          {
            token: "paren.lparen",
            regex: "[\\(]",
          },
          {
            token: "paren.rparen",
            regex: "[\\)]",
          },
          {
            token: "text",
            regex: "\\s+",
          },
        ],
      };

      this.normalizeRules();
    };

    oop.inherits(FleetHighlightRules, SqlHighlightRules);

    exports.FleetHighlightRules = FleetHighlightRules;
  }
);

ace.define(
  "ace/mode/fleet",
  [
    "require",
    "exports",
    "module",
    "ace/lib/oop",
    "ace/mode/sql",
    "ace/mode/fleet_highlight_rules",
    "ace/range",
  ],
  function (acequire, exports, module) {
    "use strict";

    var oop = acequire("../lib/oop");
    var TextMode = acequire("./sql").Mode;
    var FleetHighlightRules = acequire("./fleet_highlight_rules")
      .FleetHighlightRules;
    var Range = acequire("../range").Range;

    var Mode = function () {
      this.HighlightRules = FleetHighlightRules;
    };
    oop.inherits(Mode, TextMode);

    (function () {
      this.lineCommentStart = "--";

      this.$id = "ace/mode/fleet";
    }.call(Mode.prototype));

    exports.Mode = Mode;
  }
);
