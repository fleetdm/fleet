/* eslint-disable */
// @ts-nocheck
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
    var FleetHighlightRules = acequire("./fleet_highlight_rules").FleetHighlightRules;
    var Range = acequire("../range").Range;

    var Mode = function () {
      this.HighlightRules = FleetHighlightRules;
      // ... any additional mode setup ...
    };
    oop.inherits(Mode, TextMode);

    (function () {
      this.lineCommentStart = "--";
      this.$id = "ace/mode/fleet";
    }.call(Mode.prototype));

    exports.Mode = Mode;
  }
);
