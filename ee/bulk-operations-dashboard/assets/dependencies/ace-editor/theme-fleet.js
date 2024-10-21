ace.define(
  "ace/theme/fleet",
  ["require", "exports", "module", "ace/lib/dom"],
  function (acequire, exports, module) {
    // The CSS is inlined and backslashes are used to escape newlines
    var cssText = ".ace_editor.ace-fleet {\
  font-size: 14px;\
  background-color: #fafafa;\
  color: #66696f;\
  border-radius: 4px;\
  border: solid 1px #dbe3e5;\
  line-height: 24px;\
}\
\
.ace_editor.ace-fleet.ace_focus {\
  box-shadow: inset 0 0 6px 0 rgba(0, 0, 0, 0.16);\
  background: white;\
}\
\
.ace_editor.ace-fleet.ace_focus .ace_gutter {\
  box-shadow: 0 0 6px 0 rgba(0, 0, 0, 0.16);\
}\
.ace_editor.ace-fleet.ace_focus .ace_scroller {\
  border-bottom: solid 1px #c38dec;\
}\
\
.ace-fleet.ace_autocomplete .ace_content {\
  padding-left: 0px;\
}\
\
.ace_editor.ace-fleet.ace_autocomplete {\
  width: 350px;\
}\
\
.ace-fleet .ace_content {\
  height: 100% !important;\
}\
\
.ace-fleet .ace_gutter {\
  background: #fff;\
  color: #c38dec;\
  z-index: 1;\
  border-right: solid 1px #e3e3e3;\
}\
\
.ace-fleet .ace_gutter-active-line {\
  background-color: rgba(174, 109, 223, 0.15);\
  border-radius: 2px;\
}\
\
.ace-fleet .ace_print-margin {\
  width: 1px;\
  background: #f6f6f6;\
}\
\
.ace-fleet .ace_scrollbar {\
  z-index: 1;\
}\
\
.ace-fleet .ace_cursor {\
  color: #aeafad;\
}\
\
/* Hide cursor in read-only mode */\
.ace-fleet .ace_hidden-cursors {\
  opacity: 0;\
}\
\
.ace-fleet .ace_marker-layer .ace_selection {\
  background: rgba(74, 144, 226, 0.13);\
}\
\
.ace-fleet.ace_multiselect .ace_selection.ace_start {\
  box-shadow: 0 0 3px 0px #ffffff;\
}\
\
.ace-fleet .ace_marker-layer .ace_step {\
  background: rgb(255, 255, 0);\
}\
\
.ace-fleet .ace_marker-layer .ace_bracket {\
  margin: -1px 0 0 -1px;\
  border: 1px solid #d1d1d1;\
}\
\
.ace-fleet .ace_marker-layer .ace_selected-word {\
  border: 1px solid #d6d6d6;\
}\
\
.ace-fleet .ace_invisible {\
  color: #d1d1d1;\
}\
\
.ace-fleet .ace_keyword {\
  color: #ae6ddf;\
  font-weight: $bold;\
}\
\
.ace-fleet .ace_osquery-token {\
  border-radius: 3px;\
  background-color: #ae6ddf;\
  color: #ffffff;\
}\
\
.ace-fleet .ace_identifier {\
  color: #ff5850;\
}\
\
.ace-fleet .ace_string,\
.ace-fleet .ace_osquery-column {\
  color: #4fd061;\
}\
\
.ace-fleet .ace_meta,\
.ace-fleet .ace_storage,\
.ace-fleet .ace_storage.ace_type,\
.ace-fleet .ace_support.ace_type {\
  color: #8959a8;\
}\
\
.ace-fleet .ace_keyword.ace_operator {\
  color: #3e999f;\
}\
\
.ace-fleet .ace_constant.ace_character,\
.ace-fleet .ace_constant.ace_language,\
.ace-fleet .ace_constant.ace_numeric,\
.ace-fleet .ace_keyword.ace_other.ace_unit,\
.ace-fleet .ace_support.ace_constant,\
.ace-fleet .ace_variable.ace_parameter {\
  color: #f5871f;\
}\
\
.ace-fleet .ace_constant.ace_other {\
  color: #666969;\
}\
\
.ace-fleet .ace_invalid {\
  color: #ffffff;\
  background-color: #c82829;\
}\
\
.ace-fleet .ace_invalid.ace_deprecated {\
  color: #ffffff;\
  background-color: #ae6ddf;\
}\
\
.ace-fleet .ace_fold {\
  background-color: #4271ae;\
  border-color: #4d4d4c;\
}\
\
.ace-fleet .ace_entity.ace_name.ace_function,\
.ace-fleet .ace_support.ace_function,\
.ace-fleet .ace_variable {\
  color: #4271ae;\
}\
\
.ace-fleet .ace_support.ace_class,\
.ace-fleet .ace_support.ace_type {\
  color: #c99e00;\
}\
\
.ace-fleet .ace_heading,\
.ace-fleet .ace_markup.ace_heading,\
.ace-fleet .ace_string {\
  color: #4fd061;\
}\
\
.ace-fleet .ace_entity.ace_name.ace_tag,\
.ace-fleet .ace_entity.ace_other.ace_attribute-name,\
.ace-fleet .ace_meta.ace_tag,\
.ace-fleet .ace_string.ace_regexp,\
.ace-fleet .ace_variable {\
  color: #c82829;\
}\
\
.ace-fleet .ace_comment {\
  color: #8e908c;\
}\
\
.ace-fleet .ace_indent-guide {\
  background: url(data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAACCAYAAACZgbYnAAAAE0lEQVQImWP4////f4bdu3f/BwAlfgctduB85QAAAABJRU5ErkJggg==)\
    right repeat-y;\
}\
";

    exports.isDark = false;
    exports.cssClass = "ace-fleet";
    exports.cssText = cssText;

    var dom = acequire("../lib/dom");
    dom.importCssString(exports.cssText, exports.cssClass);
  }
);
