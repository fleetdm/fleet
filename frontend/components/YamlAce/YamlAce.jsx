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
    const { error, label } = this.props;

    const labelClassName = classnames(`${baseClass}__label`, {
      [`${baseClass}__label--error`]: error,
    });

    return <p className={labelClassName}>{error || label}</p>;
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

    const wrapperClass = classnames(wrapperClassName, {
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
