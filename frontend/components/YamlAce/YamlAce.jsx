import React, { Component } from "react";
import PropTypes from "prop-types";
import AceEditor from "react-ace";
import classnames from "classnames";

import "ace-builds/src-noconflict/mode-yaml";

const baseClass = "yaml-ace";

class YamlAce extends Component {
  static propTypes = {
    error: PropTypes.string,
    label: PropTypes.string,
    name: PropTypes.string,
    onChange: PropTypes.func.isRequired,
    value: PropTypes.string,
    wrapperClassName: PropTypes.string,
  };

  renderLabel = () => {
    const { name, error, label } = this.props;

    const labelClassName = classnames(
      `${baseClass}__label`,
      "form-field__label",
      {
        "form-field__label--error": error,
      }
    );

    return (
      <label className={labelClassName} htmlFor={name}>
        {error || label}
      </label>
    );
  };

  render() {
    const {
      label,
      name,
      onChange,
      value,
      error,
      wrapperClassName,
    } = this.props;

    const { renderLabel } = this;

    const wrapperClass = classnames(wrapperClassName, "form-field", {
      [`${baseClass}__wrapper--error`]: error,
    });

    return (
      <div className={wrapperClass}>
        {renderLabel()}
        <AceEditor
          className={baseClass}
          mode="yaml"
          theme="fleet"
          width="100%"
          minLines={2}
          maxLines={17}
          editorProps={{ $blockScrolling: Infinity }}
          value={value}
          tabSize={2}
          onChange={onChange}
          name={name}
          label={label}
        />
      </div>
    );
  }
}

export default YamlAce;
