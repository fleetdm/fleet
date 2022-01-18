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
    target: PropTypes.bool,
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
      target,
    } = this.props;

    const { renderLabel } = this;

    const onChangeFunction = () => {
      if (target) {
        onChange({ name, value });
      } else {
        onChange();
      }
    };

    const wrapperClass = classnames(wrapperClassName, {
      [`${baseClass}__wrapper--error`]: error,
    });

    console.log("name", name);
    console.log("value", value);
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
          onChange={onChangeFunction}
          name={name}
          label={label}
          target={target}
        />
      </div>
    );
  }
}

export default YamlAce;
