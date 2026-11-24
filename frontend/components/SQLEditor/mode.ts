/* eslint-disable */
// @ts-nocheck
import {
  osqueryTableNames,
  osqueryTableColumnNames,
} from "utilities/osquery_tables";
import {
  sqlBuiltinFunctions,
  sqlDataTypes,
  sqlHighlightKeywords,
} from "utilities/sql_tools";

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
      var keywords = sqlHighlightKeywords.join("|");

      var builtinConstants = "true|false";

      var builtinFunctions = sqlBuiltinFunctions.join("|");

      var dataTypes = sqlDataTypes.join("|");

      var osqueryTables = osqueryTableNames.join("|");
      var osqueryColumns = osqueryTableColumnNames.join("|");

      // Some names are both a table and a column (os_version, processes, time), so a word's class depends on
      // where it appears: right after FROM/JOIN it is a table, elsewhere the general mapper (columns win) applies.
      // Created before keywordMapper because createKeywordMapper overwrites this.$keywordList.
      var tableMapper = this.createKeywordMapper(
        { "osquery-token": osqueryTables },
        "identifier",
        true
      );

      // Any keyword other than AS/JOIN after a table reference ends the FROM clause.
      var tableClauseEndKeywords = sqlHighlightKeywords
        .filter(function (kw) {
          return kw !== "as" && kw !== "join";
        })
        .join("|");

      var identifierRegex = "[a-zA-Z_$][a-zA-Z0-9_$]*\\b";

      var commentRules = function () {
        return [
          {
            token: "comment",
            regex: "--.*$",
          },
          {
            token: "comment",
            start: "/\\*",
            end: "\\*/",
          },
        ];
      };

      // Parentheses push onto Ace's state stack so the ")" closing a FROM-clause subquery resumes the FROM clause.
      var pushParen = function (returnState) {
        return function (currentState, stack) {
          stack.unshift("start", returnState);
          return "start";
        };
      };
      var popParen = function (currentState, stack) {
        stack.shift();
        return stack.shift() || "start";
      };

      var keywordMapper = this.createKeywordMapper(
        {
          "osquery-token": osqueryTables,
          "osquery-column": osqueryColumns,
          "support.function": builtinFunctions,
          keyword: keywords,
          "constant.language": builtinConstants,
          "storage.type": dataTypes,
        },
        "identifier",
        true
      );

      this.$rules = {
        start: commentRules().concat([
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
            token: "keyword",
            regex: "\\b(?:from|join)\\b",
            caseInsensitive: true,
            next: "tableRef",
          },
          {
            token: keywordMapper,
            regex: identifierRegex,
          },
          {
            token: "keyword.operator",
            regex:
              "\\+|\\-|\\/|\\/\\/|%|<@>|@>|<@|&|\\^|~|<|>|<=|=>|==|!=|<>|=",
          },
          {
            token: "paren.lparen",
            regex: "[\\(]",
            next: pushParen("start"),
          },
          {
            token: "paren.rparen",
            regex: "[\\)]",
            next: popParen,
          },
          {
            token: "text",
            regex: "\\s+",
          },
        ]),
        // Expects a table name, or "(" for a subquery.
        tableRef: commentRules().concat([
          {
            token: "text",
            regex: "\\s+",
          },
          {
            token: "paren.lparen",
            regex: "[\\(]",
            next: pushParen("afterTable"),
          },
          {
            // Schema prefix, as in main.os_version
            token: "identifier",
            regex: identifierRegex + "(?=\\.)",
          },
          {
            token: "text",
            regex: "\\.",
          },
          {
            token: tableMapper,
            regex: identifierRegex,
            next: "afterTable",
          },
          {
            token: "empty",
            regex: "(?=\\S)",
            next: "start",
          },
        ]),
        // After a table name: an optional alias, then "," or JOIN for another table, or a keyword ending the clause.
        afterTable: commentRules().concat([
          {
            token: "text",
            regex: "\\s+",
          },
          {
            token: "text",
            regex: ",",
            next: "tableRef",
          },
          {
            token: "keyword",
            regex: "\\bjoin\\b",
            caseInsensitive: true,
            next: "tableRef",
          },
          {
            token: keywordMapper,
            regex: "\\b(?:" + tableClauseEndKeywords + ")\\b",
            caseInsensitive: true,
            next: "start",
          },
          {
            token: keywordMapper,
            regex: "\\bas\\b",
            caseInsensitive: true,
          },
          {
            token: "identifier",
            regex: identifierRegex,
          },
          {
            token: "empty",
            regex: "(?=\\S)",
            next: "start",
          },
        ]),
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
