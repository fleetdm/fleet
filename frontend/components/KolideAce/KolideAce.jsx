import React, { Component } from "react";
import PropTypes from "prop-types";
import AceEditor from "react-ace";
import classnames from "classnames";
import "brace/mode/sql";
import "brace/ext/linking";
import "brace/ext/language_tools";

import "./mode";
import "./theme";

const baseClass = "kolide-ace";

class KolideAce extends Component {
  static propTypes = {
    error: PropTypes.string,
    fontSize: PropTypes.number,
    handleSubmit: PropTypes.func.isRequired,
    label: PropTypes.string,
    name: PropTypes.string,
    onChange: PropTypes.func.isRequired,
    onLoad: PropTypes.func.isRequired,
    value: PropTypes.string,
    readOnly: PropTypes.bool,
    showGutter: PropTypes.bool,
    wrapEnabled: PropTypes.bool,
    wrapperClassName: PropTypes.string,
    hint: PropTypes.string,
  };

  static defaultProps = {
    fontSize: 14,
    name: "query-editor",
    showGutter: true,
    wrapEnabled: false,
  };

  renderLabel = () => {
    const { error, label } = this.props;

    const labelClassName = classnames(`${baseClass}__label`, {
      [`${baseClass}__label--error`]: error,
    });

    return <p className={labelClassName}>{error || label}</p>;
  };

  renderHint = () => {
    const { hint } = this.props;

    if (hint) {
      return <span className={`${baseClass}__hint`}>{hint}</span>;
    }

    return false;
  };

  render() {
    const {
      error,
      fontSize,
      handleSubmit,
      name,
      onChange,
      onLoad,
      readOnly,
      showGutter,
      value,
      wrapEnabled,
      wrapperClassName,
    } = this.props;
    const { renderLabel, renderHint } = this;

    const wrapperClass = classnames(wrapperClassName, {
      [`${baseClass}__wrapper--error`]: error,
    });

    const fixHotkeys = (editor) => {
      editor.commands.removeCommands(["gotoline", "find"]);
      onLoad && onLoad(editor);
    };

    return (
      <div className={wrapperClass}>
        {renderLabel()}
        <AceEditor
          enableBasicAutocompletion
          enableLiveAutocompletion
          editorProps={{ $blockScrolling: Infinity }}
          fontSize={fontSize}
          mode="kolide"
          minLines={2}
          maxLines={20}
          name={name}
          onChange={onChange}
          onLoad={fixHotkeys}
          readOnly={readOnly}
          setOptions={{ enableLinking: true }}
          showGutter={showGutter}
          showPrintMargin={false}
          theme="kolide"
          value={value}
          width="100%"
          wrapEnabled={wrapEnabled}
          commands={[
            {
              name: "commandName",
              bindKey: { win: "Ctrl-Enter", mac: "Ctrl-Enter" },
              exec: handleSubmit,
            },
          ]}
        />
        {renderHint()}
      </div>
    );
  }
}

export default KolideAce;
