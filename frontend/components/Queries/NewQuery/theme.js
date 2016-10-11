/* eslint-disable */
ace.define("ace/theme/kolide",["require","exports","module","ace/lib/dom"], function(acequire, exports, module) {

  exports.isDark = false;
  exports.cssClass = "ace-kolide";
  exports.cssText = ".ace-kolide .ace_gutter {\
                     background: #9CA3AC;\
                     color: #FFF\
}\
.ace-kolide .ace_print-margin {\
  width: 1px;\
    background: #f6f6f6\
}\
.ace-kolide .ace_osquery-token{\
  background-color: #AE6DDF;\
    color: #FFFFFF\
}\
.ace-kolide {\
  background-color: #EAEDFB;\
    color: #4D4D4C\
}\
.ace-kolide .ace_cursor {\
  color: #AEAFAD\
}\
.ace-kolide .ace_marker-layer .ace_selection {\
  background: #D6D6D6\
}\
  .ace-kolide.ace_multiselect .ace_selection.ace_start {\
    box-shadow: 0 0 3px 0px #FFFFFF;\
  }\
  .ace-kolide .ace_marker-layer .ace_step {\
    background: rgb(255, 255, 0)\
  }\
  .ace-kolide .ace_marker-layer .ace_bracket {\
    margin: -1px 0 0 -1px;\
      border: 1px solid #D1D1D1\
  }\
  .ace-kolide .ace_marker-layer .ace_active-line {\
    background: #EFEFEF\
  }\
  .ace-kolide .ace_gutter-active-line {\
    background-color : #9CA3AC\
  }\
  .ace-kolide .ace_marker-layer .ace_selected-word {\
    border: 1px solid #D6D6D6\
  }\
  .ace-kolide .ace_invisible {\
    color: #D1D1D1\
  }\
  .ace-kolide .ace_keyword,\
  .ace-kolide .ace_meta,\
  .ace-kolide .ace_storage,\
  .ace-kolide .ace_storage.ace_type,\
  .ace-kolide .ace_support.ace_type {\
    color: #8959A8\
  }\
  .ace-kolide .ace_keyword.ace_operator {\
    color: #3E999F\
  }\
  .ace-kolide .ace_constant.ace_character,\
  .ace-kolide .ace_constant.ace_language,\
  .ace-kolide .ace_constant.ace_numeric,\
  .ace-kolide .ace_keyword.ace_other.ace_unit,\
  .ace-kolide .ace_support.ace_constant,\
  .ace-kolide .ace_variable.ace_parameter {\
    color: #F5871F\
  }\
  .ace-kolide .ace_constant.ace_other {\
    color: #666969\
  }\
  .ace-kolide .ace_invalid {\
    color: #FFFFFF;\
      background-color: #C82829\
  }\
  .ace-kolide .ace_invalid.ace_deprecated {\
    color: #FFFFFF;\
      background-color: #AE6DDF\
  }\
  .ace-kolide .ace_fold {\
    background-color: #4271AE;\
      border-color: #4D4D4C\
  }\
  .ace-kolide .ace_entity.ace_name.ace_function,\
  .ace-kolide .ace_support.ace_function,\
  .ace-kolide .ace_variable {\
    color: #4271AE\
  }\
  .ace-kolide .ace_support.ace_class,\
  .ace-kolide .ace_support.ace_type {\
    color: #C99E00\
  }\
  .ace-kolide .ace_heading,\
  .ace-kolide .ace_markup.ace_heading,\
  .ace-kolide .ace_string {\
    color: #4FD061\
  }\
  .ace-kolide .ace_entity.ace_name.ace_tag,\
  .ace-kolide .ace_entity.ace_other.ace_attribute-name,\
  .ace-kolide .ace_meta.ace_tag,\
  .ace-kolide .ace_string.ace_regexp,\
  .ace-kolide .ace_variable {\
    color: #C82829\
  }\
  .ace-kolide .ace_comment {\
    color: #8E908C\
  }\
  .ace-kolide .ace_indent-guide {\
    background: url(data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAACCAYAAACZgbYnAAAAE0lEQVQImWP4////f4bdu3f/BwAlfgctduB85QAAAABJRU5ErkJggg==) right repeat-y\
  }";

var dom = acequire("../lib/dom");
dom.importCssString(exports.cssText, exports.cssClass);
});
