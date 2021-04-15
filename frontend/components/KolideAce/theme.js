/* eslint-disable */
ace.define(
  "ace/theme/kolide",
  ["require", "exports", "module", "ace/lib/dom"],
  function (acequire, exports, module) {
    exports.isDark = false;
    exports.cssClass = "ace-kolide";
    exports.cssText = require("./theme.css");

    var dom = acequire("../lib/dom");
    dom.importCssString(exports.cssText, exports.cssClass);
  }
);
